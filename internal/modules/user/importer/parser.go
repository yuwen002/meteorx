// Package importer 负责把用户批量导入文件（CSV / XLSX）解析为结构化行。
// 本包只处理"格式层"的事情——定位表头、映射列、跳过空行、限制规模；
// 字段级业务校验（长度/格式/唯一性/角色）由 service 层统一完成，
// 这样两种文件格式共享同一套校验与错误文案。
package importer

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/xuri/excelize/v2"
)

const (
	// MaxRows 单次导入的行数上限（不含表头）。bcrypt 哈希与单事务写入的成本随行数线性增长，
	// 500 行在 8 并发哈希下约 2~4 秒，是"同步接口"能接受的上限。
	MaxRows = 500
	// MaxFileSize 上传文件大小上限。
	MaxFileSize = 5 << 20
)

// 列名常量（内部统一标识，避免各处散落字符串）。
const (
	colUsername   = "username"
	colNickname   = "nickname"
	colEmail      = "email"
	colDepartment = "department"
	colPosition   = "position"
	colRoles      = "roles"
)

// Row 是文件中一行解析后的结果。LineNo 为文件内的原始行号（1 起，表头为第 1 行），
// 用于把校验错误精确回传给操作者。
type Row struct {
	LineNo     int
	Username   string
	Nickname   string
	Email      string
	Department string
	Position   string
	RoleCodes  []string
}

// headerAliases 表头别名：英文列名与常见中文列名都接受，大小写不敏感。
// 只识别已知列，未知列会被忽略并在 warnings 中提示，避免用户以为该列生效了。
var headerAliases = map[string]string{
	"username":   colUsername,
	"用户名":        colUsername,
	"登录名":        colUsername,
	"账号":         colUsername,
	"nickname":   colNickname,
	"昵称":         colNickname,
	"姓名":         colNickname,
	"email":      colEmail,
	"邮箱":         colEmail,
	"电子邮件":       colEmail,
	"department": colDepartment,
	"部门":         colDepartment,
	"position":   colPosition,
	"职位":         colPosition,
	"岗位":         colPosition,
	"roles":      colRoles,
	"角色":         colRoles,
	"角色编码":       colRoles,
}

// 文件级错误哨兵，handler 据此返回 400 而不是 500。
var (
	ErrEmptyFile      = errors.New("文件内容为空")
	ErrUnsupportedExt = errors.New("仅支持 .csv 或 .xlsx 文件")
	ErrTooManyRows    = fmt.Errorf("单次最多导入 %d 行", MaxRows)
	ErrMissingHeader  = errors.New("表头缺少必填列")
)

// Parse 按扩展名解析文件内容，返回数据行与告警信息。
// 返回的 error 一定是"整个文件不可用"级别的问题；单行问题不在此处报错。
func Parse(filename string, data []byte) ([]Row, []string, error) {
	if len(bytes.TrimSpace(data)) == 0 {
		return nil, nil, ErrEmptyFile
	}

	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".csv":
		return parseCSV(data)
	case ".xlsx":
		return parseXLSX(data)
	case "":
		// 某些客户端不带扩展名，按 CSV 兜底（CSV 是纯文本，误判成本低）
		return parseCSV(data)
	default:
		return nil, nil, ErrUnsupportedExt
	}
}

// parseCSV 解析 CSV：兼容 UTF-8 BOM、CRLF、列数不齐与含引号字段。
func parseCSV(data []byte) ([]Row, []string, error) {
	data = bytes.TrimPrefix(data, []byte{0xEF, 0xBB, 0xBF}) // 去除 UTF-8 BOM（Excel 另存 CSV 常见）

	reader := csv.NewReader(bytes.NewReader(data))
	reader.FieldsPerRecord = -1 // 允许每行列数不一致，由我们按表头长度对齐
	reader.LazyQuotes = true
	reader.TrimLeadingSpace = true

	var records [][]string
	var lines []int // records[i] 对应的文件物理行号（1 起）
	for {
		rec, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, nil, fmt.Errorf("CSV 解析失败（第 %d 行）：%v", len(records)+1, err)
		}
		// FieldPos 给出当前记录首列所在的物理行号：encoding/csv 会静默跳过空行，
		// 用记录下标当行号会在用户文件含空行时错位，这里必须取真实行号。
		line, _ := reader.FieldPos(0)
		records = append(records, rec)
		lines = append(lines, line)
	}
	return buildRows(records, lines)
}

// parseXLSX 解析 XLSX 的第一个工作表。
func parseXLSX(data []byte) ([]Row, []string, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, nil, fmt.Errorf("XLSX 解析失败：%v", err)
	}
	defer func() { _ = f.Close() }()

	sheet := f.GetSheetName(0)
	if sheet == "" {
		return nil, nil, ErrEmptyFile
	}
	records, err := f.GetRows(sheet)
	if err != nil {
		return nil, nil, fmt.Errorf("读取工作表失败：%v", err)
	}
	// XLSX 的行号即工作表行号，用 nil 让 buildRows 按下标推导
	return buildRows(records, nil)
}

// buildRows 把二维单元格文本映射为 Row 列表（CSV 与 XLSX 共用）。
// lines 非空时与 records 一一对应，给出每行在文件中的物理行号；为空时按下标推导。
func buildRows(records [][]string, lines []int) ([]Row, []string, error) {
	// 跳过表头之前可能存在的完全空行（用户常在首行留白）
	headerIdx := -1
	for i, rec := range records {
		if !isBlankRecord(rec) {
			headerIdx = i
			break
		}
	}
	if headerIdx == -1 {
		return nil, nil, ErrEmptyFile
	}

	colIndex, warnings, err := resolveHeader(records[headerIdx])
	if err != nil {
		return nil, nil, err
	}

	rows := make([]Row, 0, len(records)-headerIdx-1)
	for i := headerIdx + 1; i < len(records); i++ {
		rec := records[i]
		if isBlankRecord(rec) {
			continue
		}
		rows = append(rows, Row{
			LineNo:     lineNoAt(lines, i), // 文件内行号（1 起）
			Username:   cellAt(rec, colIndex[colUsername]),
			Nickname:   cellAt(rec, colIndex[colNickname]),
			Email:      cellAt(rec, colIndex[colEmail]),
			Department: cellAt(rec, colIndex[colDepartment]),
			Position:   cellAt(rec, colIndex[colPosition]),
			RoleCodes:  splitRoleCodes(cellAt(rec, colIndex[colRoles])),
		})
	}

	if len(rows) == 0 {
		return nil, warnings, ErrEmptyFile
	}
	if len(rows) > MaxRows {
		return nil, warnings, fmt.Errorf("%w，当前 %d 行，请拆分后再导入", ErrTooManyRows, len(rows))
	}
	return rows, warnings, nil
}

// lineNoAt 返回 records[idx] 的文件行号。
func lineNoAt(lines []int, idx int) int {
	if idx < len(lines) {
		return lines[idx]
	}
	return idx + 1
}

// resolveHeader 解析表头，返回 列名 → 列下标 的映射。
// 必填列缺失时直接失败（整份文件不可用）；未知列进 warnings。
func resolveHeader(header []string) (map[string]int, []string, error) {
	colIndex := make(map[string]int, len(headerAliases))
	var unknown []string

	for i, raw := range header {
		name := strings.ToLower(strings.TrimSpace(strings.TrimPrefix(raw, "\xEF\xBB\xBF")))
		if name == "" {
			continue
		}
		key, ok := headerAliases[name]
		if !ok {
			unknown = append(unknown, strings.TrimSpace(raw))
			continue
		}
		if _, dup := colIndex[key]; dup {
			continue // 重复列取第一处
		}
		colIndex[key] = i
	}

	missing := make([]string, 0, 2)
	if _, ok := colIndex[colUsername]; !ok {
		missing = append(missing, "用户名（username）")
	}
	if _, ok := colIndex[colNickname]; !ok {
		missing = append(missing, "昵称（nickname）")
	}
	if len(missing) > 0 {
		return nil, nil, fmt.Errorf("%w：%s", ErrMissingHeader, strings.Join(missing, "、"))
	}

	var warnings []string
	if len(unknown) > 0 {
		warnings = append(warnings, fmt.Sprintf("已忽略无法识别的列：%s", strings.Join(unknown, "、")))
	}
	return colIndex, warnings, nil
}

// cellAt 安全取单元格（列数不齐时返回空串）。
func cellAt(rec []string, idx int) string {
	if idx < 0 || idx >= len(rec) {
		return ""
	}
	return strings.TrimSpace(rec[idx])
}

// isBlankRecord 判断整行是否全为空单元格。
func isBlankRecord(rec []string) bool {
	for _, c := range rec {
		if strings.TrimSpace(c) != "" {
			return false
		}
	}
	return true
}

// splitRoleCodes 拆分角色编码列：支持中英文逗号、分号、竖线与空白分隔，自动去重去空。
func splitRoleCodes(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	fields := strings.FieldsFunc(raw, func(r rune) bool {
		switch r {
		case ',', ';', '|', '，', '；', '、', ' ', '\t':
			return true
		}
		return false
	})
	seen := make(map[string]struct{}, len(fields))
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		code := strings.TrimSpace(f)
		if code == "" {
			continue
		}
		if _, dup := seen[code]; dup {
			continue
		}
		seen[code] = struct{}{}
		out = append(out, code)
	}
	return out
}
