package exporter

import (
	"encoding/csv"
	"io"
)

// csvSink 以纯 CSV 写入（不含 BOM，由调用方按需写入 UTF-8 BOM 以保证 Excel 中文不乱码）。
// 每批 WriteRows 后立即 Flush，保持大数据量下的流式内存占用。
type csvSink struct {
	w *csv.Writer
}

// NewCSVSink 创建 CSV 行接收器
func NewCSVSink(dst io.Writer) RowSink {
	return &csvSink{w: csv.NewWriter(dst)}
}

func (s *csvSink) WriteHeader(columns []string) error {
	if err := s.w.Write(columns); err != nil {
		return err
	}
	s.w.Flush()
	return s.w.Error()
}

func (s *csvSink) WriteRows(rows [][]string) (int64, error) {
	if len(rows) == 0 {
		return 0, nil
	}
	if err := s.w.WriteAll(rows); err != nil {
		return 0, err
	}
	s.w.Flush()
	if err := s.w.Error(); err != nil {
		return 0, err
	}
	return int64(len(rows)), nil
}

func (s *csvSink) Close() error {
	s.w.Flush()
	return s.w.Error()
}
