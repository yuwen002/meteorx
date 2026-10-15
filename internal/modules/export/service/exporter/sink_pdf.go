package exporter

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/jung-kurt/gofpdf"
)

// PDF 导出需要可渲染中文的 CJK 字体（gofpdf 核心字体不支持中文）。
// METEORX_EXPORT_PDF_FONT 指向一个 .ttf 字体文件即可启用；否则导出任务会明确报错，
// 绝不输出中文乱码。这是生产环境的标准做法（内嵌字体属于运维/合规决策）。
var (
	pdfFontPath string
	pdfFontName string
)

// SetPDFFont 注入 PDF 导出使用的 CJK 字体（path 为 .ttf 文件路径，name 为字体族名）。
func SetPDFFont(path, name string) {
	pdfFontPath = path
	if name == "" {
		name = "CJK"
	}
	pdfFontName = name
}

// detectPDFFont 解析可用的 CJK 字体：优先环境变量，其次启动时注入的路径，最后常见系统路径。
func detectPDFFont() (string, string) {
	if p := strings.TrimSpace(os.Getenv("METEORX_EXPORT_PDF_FONT")); p != "" {
		name := strings.TrimSpace(os.Getenv("METEORX_EXPORT_PDF_FONT_NAME"))
		if name == "" {
			name = "CJK"
		}
		return p, name
	}
	if pdfFontPath != "" {
		name := pdfFontName
		if name == "" {
			name = "CJK"
		}
		return pdfFontPath, name
	}
	// 只探 .ttf 单文件字体：gofpdf 不支持 .ttc 字体集合，探中只会导出失败，不如直接提示配置
	candidates := []string{
		`C:\Windows\Fonts\msyh.ttf`,   // Windows 微软雅黑（单一 ttf）
		`C:\Windows\Fonts\simsun.ttf`, // Windows 宋体
		"/usr/share/fonts/truetype/noto/NotoSansSC-Regular.ttf",
		"/usr/share/fonts/truetype/wqy/wqy-zenhei.ttf",
		"/Library/Fonts/Arial Unicode.ttf",
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c, "CJK"
		}
	}
	return "", ""
}

const (
	pdfMargin   = 8.0
	pdfLineH    = 5.0
	pdfFontSize = 8.0
	pdfCharMM   = 2.4 // 每字符约占用宽度（mm），用于估算换行行数（偏保守）
)

// pdfSink 用 gofpdf 将表头与数据行渲染为带边框的表格 PDF。
type pdfSink struct {
	pdf        gofpdf.Pdf
	dst        io.Writer
	fontName   string
	cols       []string
	colW       []float64
	usableW    float64
	pageBottom float64
}

// NewPDFSink 创建 PDF 行接收器；缺少可用 CJK 字体时返回 error。
func NewPDFSink(dst io.Writer) (RowSink, error) {
	path, name := detectPDFFont()
	if path == "" {
		return nil, fmt.Errorf("PDF 导出需要 CJK 字体：请设置环境变量 METEORX_EXPORT_PDF_FONT 指向一个 .ttf 字体文件")
	}
	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.AddUTF8Font(name, "", path)
	if pdf.Err() {
		return nil, fmt.Errorf("加载 PDF 字体失败（需单一 .ttf 文件，.ttc 集合不被支持）")
	}
	pdf.SetAutoPageBreak(false, 0)
	pdf.SetMargins(pdfMargin, pdfMargin, pdfMargin)
	pdf.AddPage()
	pdf.SetFont(name, "", pdfFontSize)
	return &pdfSink{
		pdf:        pdf,
		dst:        dst,
		fontName:   name,
		usableW:    210 - 2*pdfMargin,
		pageBottom: 297 - 12,
	}, nil
}

func (s *pdfSink) computeColWidths(cols []string) []float64 {
	n := len(cols)
	if n == 0 {
		return nil
	}
	w := make([]float64, n)
	base := s.usableW / float64(n)
	for i := range w {
		w[i] = base
	}
	return w
}

func (s *pdfSink) charsPerLine(i int) int {
	if i >= len(s.colW) {
		return 1
	}
	cpl := int(s.colW[i] / pdfCharMM)
	if cpl < 1 {
		cpl = 1
	}
	return cpl
}

// renderRow 渲染一行（header=true 时加粗并填充表头底色），自动处理分页并在新页重绘表头。
func (s *pdfSink) renderRow(row []string, header bool) {
	if s.pdf.GetY()+pdfLineH > s.pageBottom {
		s.pdf.AddPage()
		s.pdf.SetFont(s.fontName, "B", pdfFontSize)
		s.renderRow(s.cols, true)
		s.pdf.SetFont(s.fontName, "", pdfFontSize)
	}
	x0 := s.pdf.GetX()
	y0 := s.pdf.GetY()

	maxLines := 1
	for i, v := range row {
		if i >= len(s.colW) {
			continue
		}
		cpl := s.charsPerLine(i)
		runes := []rune(v)
		lines := 1
		if len(runes) > cpl {
			lines = (len(runes) + cpl - 1) / cpl
		}
		if lines > maxLines {
			maxLines = lines
		}
	}
	cellH := float64(maxLines) * pdfLineH

	if header {
		s.pdf.SetFont(s.fontName, "B", pdfFontSize)
		s.pdf.SetFillColor(232, 236, 243)
	} else {
		s.pdf.SetFont(s.fontName, "", pdfFontSize)
		s.pdf.SetFillColor(255, 255, 255)
	}
	s.pdf.SetDrawColor(200, 200, 200)

	x := x0
	for i, v := range row {
		if i >= len(s.colW) {
			break
		}
		w := s.colW[i]
		s.pdf.Rect(x, y0, w, cellH, "D")
		s.pdf.SetXY(x, y0)
		s.pdf.MultiCell(w, pdfLineH, v, "", "L", header)
		s.pdf.SetXY(x+w, y0)
		x += w
	}
	s.pdf.SetXY(x0, y0+cellH)
}

func (s *pdfSink) WriteHeader(columns []string) error {
	s.cols = columns
	s.colW = s.computeColWidths(columns)
	s.renderRow(columns, true)
	s.pdf.SetFont(s.fontName, "", pdfFontSize)
	return nil
}

func (s *pdfSink) WriteRows(rows [][]string) (int64, error) {
	for _, r := range rows {
		s.renderRow(r, false)
	}
	return int64(len(rows)), nil
}

func (s *pdfSink) Close() error {
	return s.pdf.Output(s.dst)
}
