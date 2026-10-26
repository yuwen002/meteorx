package importer

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestParseCSV_EnglishHeader(t *testing.T) {
	data := "username,nickname,email,department,position,roles\n" +
		"alice,Alice Chen,alice@example.com,研发中心,工程师,member\n" +
		"bob,Bob Li,,,,\n"

	rows, warnings, err := Parse("users.csv", []byte(data))
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("期望无告警，实际 %v", warnings)
	}
	if len(rows) != 2 {
		t.Fatalf("期望 2 行，实际 %d", len(rows))
	}

	first := rows[0]
	if first.LineNo != 2 {
		t.Errorf("首行 LineNo = %d，期望 2", first.LineNo)
	}
	if first.Username != "alice" || first.Nickname != "Alice Chen" {
		t.Errorf("首行解析结果异常: %+v", first)
	}
	if first.Email != "alice@example.com" || first.Department != "研发中心" || first.Position != "工程师" {
		t.Errorf("首行可选列解析异常: %+v", first)
	}
	if len(first.RoleCodes) != 1 || first.RoleCodes[0] != "member" {
		t.Errorf("角色解析异常: %v", first.RoleCodes)
	}

	second := rows[1]
	if second.Email != "" || second.Department != "" || len(second.RoleCodes) != 0 {
		t.Errorf("空单元格应解析为空值: %+v", second)
	}
}

func TestParseCSV_ChineseHeaderAndRoleSplit(t *testing.T) {
	data := "用户名,昵称,邮箱,部门,岗位,角色编码\n" +
		"carol,卡罗,,,,\"member; viewer，auditor\"\n"

	rows, _, err := Parse("用户导入.csv", []byte(data))
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("期望 1 行，实际 %d", len(rows))
	}
	got := rows[0].RoleCodes
	want := []string{"member", "viewer", "auditor"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("角色拆分 = %v，期望 %v", got, want)
	}
}

func TestParseCSV_BOMAndCRLF(t *testing.T) {
	data := "\xEF\xBB\xBFusername,nickname\r\ndave,Dave\r\n"

	rows, _, err := Parse("bom.csv", []byte(data))
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if len(rows) != 1 || rows[0].Username != "dave" {
		t.Fatalf("BOM 处理异常: %+v", rows)
	}
}

func TestParseCSV_SkipsBlankRowsAndLeadingBlank(t *testing.T) {
	data := "\n\nusername,nickname\n" +
		"erin,Erin\n" +
		",\n" +
		"frank,Frank\n"

	rows, _, err := Parse("blank.csv", []byte(data))
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("期望跳过空行后 2 行，实际 %d", len(rows))
	}
	// 表头位于第 3 行，因此数据从第 4、6 行来（第 5 行整行为空被跳过）
	if rows[0].LineNo != 4 || rows[1].LineNo != 6 {
		t.Errorf("行号映射异常: %d, %d", rows[0].LineNo, rows[1].LineNo)
	}
}

func TestParseCSV_WarnsUnknownColumns(t *testing.T) {
	data := "username,nickname,手机号,备注\n" +
		"grace,Grace,13800000000,x\n"

	rows, warnings, err := Parse("unknown.csv", []byte(data))
	if err != nil {
		t.Fatalf("解析失败: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("期望 1 行，实际 %d", len(rows))
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "手机号") {
		t.Errorf("未知列告警缺失: %v", warnings)
	}
}

func TestParseCSV_MissingRequiredHeader(t *testing.T) {
	data := "username,email\nalice,a@example.com\n"

	_, _, err := Parse("missing.csv", []byte(data))
	if err == nil {
		t.Fatal("缺少昵称列应报错")
	}
	if !errors.Is(err, ErrMissingHeader) {
		t.Fatalf("错误类型不符: %v", err)
	}
	if !strings.Contains(err.Error(), "昵称") {
		t.Errorf("错误消息应指明缺失列: %v", err)
	}
}

func TestParseCSV_EmptyFile(t *testing.T) {
	for _, data := range []string{"", "   \n\n", "username,nickname\n"} {
		if _, _, err := Parse("empty.csv", []byte(data)); !errors.Is(err, ErrEmptyFile) {
			t.Errorf("输入 %q 应返回 ErrEmptyFile，实际 %v", data, err)
		}
	}
}

func TestParseCSV_TooManyRows(t *testing.T) {
	var b strings.Builder
	b.WriteString("username,nickname\n")
	for i := 0; i <= MaxRows; i++ {
		fmt.Fprintf(&b, "user%d,User%d\n", i, i)
	}

	_, _, err := Parse("big.csv", []byte(b.String()))
	if !errors.Is(err, ErrTooManyRows) {
		t.Fatalf("应返回 ErrTooManyRows，实际 %v", err)
	}
	if !strings.Contains(err.Error(), fmt.Sprintf("%d", MaxRows+1)) {
		t.Errorf("错误消息应包含实际行数: %v", err)
	}
}

func TestParse_UnsupportedExtension(t *testing.T) {
	if _, _, err := Parse("users.txt", []byte("a,b")); !errors.Is(err, ErrUnsupportedExt) {
		t.Fatalf("应返回 ErrUnsupportedExt，实际 %v", err)
	}
}

func TestParseXLSX_FirstSheet(t *testing.T) {
	f := excelize.NewFile()
	defer func() { _ = f.Close() }()

	sheet := f.GetSheetName(0)
	values := [][]any{
		{"username", "nickname", "email", "roles"},
		{"henry", "Henry", "henry@example.com", "member; viewer"},
		{"iris", "Iris", "", ""},
	}
	for r, row := range values {
		for c, v := range row {
			cell, err := excelize.CoordinatesToCellName(c+1, r+1)
			if err != nil {
				t.Fatalf("坐标转换失败: %v", err)
			}
			if err := f.SetCellValue(sheet, cell, v); err != nil {
				t.Fatalf("写入单元格失败: %v", err)
			}
		}
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		t.Fatalf("生成 xlsx 失败: %v", err)
	}

	rows, warnings, err := Parse("users.xlsx", buf.Bytes())
	if err != nil {
		t.Fatalf("解析 xlsx 失败: %v", err)
	}
	if len(warnings) != 0 {
		t.Errorf("期望无告警，实际 %v", warnings)
	}
	if len(rows) != 2 {
		t.Fatalf("期望 2 行，实际 %d", len(rows))
	}
	if rows[0].Username != "henry" || rows[0].LineNo != 2 {
		t.Errorf("首行解析异常: %+v", rows[0])
	}
	if strings.Join(rows[0].RoleCodes, ",") != "member,viewer" {
		t.Errorf("角色解析异常: %v", rows[0].RoleCodes)
	}
}

func TestParseXLSX_InvalidContent(t *testing.T) {
	if _, _, err := Parse("broken.xlsx", []byte("not a real xlsx")); err == nil {
		t.Fatal("非法 xlsx 内容应报错")
	}
}
