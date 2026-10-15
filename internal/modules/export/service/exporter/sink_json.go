package exporter

import (
	"encoding/json"
	"fmt"
	"io"
)

// jsonSink 将表头与数据行序列化为 JSON 数组：每个数据行是一个以表头为键的对象。
// 流式写出：WriteHeader 写入 "["，每个 WriteRows 内逐行写入对象并以逗号分隔，Close 写入 "]"。
type jsonSink struct {
	w      io.Writer
	enc    *json.Encoder
	header []string
	first  bool
}

// NewJSONSink 创建 JSON 行接收器
func NewJSONSink(dst io.Writer) RowSink {
	return &jsonSink{w: dst, enc: json.NewEncoder(dst), first: true}
}

func (s *jsonSink) WriteHeader(columns []string) error {
	s.header = columns
	_, err := s.w.Write([]byte("["))
	return err
}

func (s *jsonSink) WriteRows(rows [][]string) (int64, error) {
	// 未写表头时 "[" 尚未输出，继续写入会产生非法 JSON，提前拒绝
	if s.header == nil {
		return 0, fmt.Errorf("json sink: WriteHeader must be called before WriteRows")
	}
	for _, row := range rows {
		obj := make(map[string]string, len(s.header))
		for i, h := range s.header {
			if i < len(row) {
				obj[h] = row[i]
			}
		}
		if !s.first {
			if _, err := s.w.Write([]byte(",")); err != nil {
				return 0, err
			}
		}
		s.first = false
		if err := s.enc.Encode(obj); err != nil {
			return 0, err
		}
	}
	return int64(len(rows)), nil
}

func (s *jsonSink) Close() error {
	if s.header == nil {
		return fmt.Errorf("json sink: WriteHeader must be called before Close")
	}
	_, err := s.w.Write([]byte("]"))
	return err
}
