package exporter

import (
	"io"

	"github.com/xuri/excelize/v2"
)

// xlsxSink 使用 excelize 的 StreamWriter 将表头与数据行写入 xlsx 工作表（Sheet1）。
// StreamWriter 分批写入 zip，内存占用低，适合大数据量导出。
type xlsxSink struct {
	f    *excelize.File
	sw   *excelize.StreamWriter
	dst  io.Writer
	row  int
}

// NewXLSXSink 创建 XLSX 行接收器
func NewXLSXSink(dst io.Writer) (RowSink, error) {
	f := excelize.NewFile()
	sw, err := f.NewStreamWriter("Sheet1")
	if err != nil {
		return nil, err
	}
	return &xlsxSink{f: f, sw: sw, dst: dst, row: 1}, nil
}

func (s *xlsxSink) writeRow(values []string) error {
	cells := make([]interface{}, len(values))
	for i, v := range values {
		cells[i] = v
	}
	ax, err := excelize.CoordinatesToCellName(1, s.row)
	if err != nil {
		return err
	}
	if err := s.sw.SetRow(ax, cells); err != nil {
		return err
	}
	s.row++
	return nil
}

func (s *xlsxSink) WriteHeader(columns []string) error {
	return s.writeRow(columns)
}

func (s *xlsxSink) WriteRows(rows [][]string) (int64, error) {
	for _, r := range rows {
		if err := s.writeRow(r); err != nil {
			return 0, err
		}
	}
	return int64(len(rows)), nil
}

func (s *xlsxSink) Close() error {
	if err := s.sw.Flush(); err != nil {
		return err
	}
	return s.f.Write(s.dst)
}
