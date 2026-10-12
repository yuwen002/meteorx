package exporter

import (
	"encoding/csv"
	"time"
)

// timeLayout 导出 CSV 中时间列的统一格式
const timeLayout = "2006-01-02 15:04:05"

// exportBatchSize FindInBatches 单批读取行数，兼顾内存占用与查询往返次数
const exportBatchSize = 500

// filterString 从筛选条件中取字符串值，缺失或类型不符返回空串
func filterString(filters map[string]any, key string) string {
	if filters == nil {
		return ""
	}
	if v, ok := filters[key].(string); ok {
		return v
	}
	return ""
}

// fmtTime 格式化非空时间
func fmtTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format(timeLayout)
}

// fmtNullTime 格式化可空时间，nil 返回空串
func fmtNullTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return fmtTime(*t)
}

// fmtStr 解引用可空字符串
func fmtStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// fmtInt 格式化可空整型
func fmtInt(v *int) string {
	if v == nil {
		return ""
	}
	return itoa(*v)
}

// itoa 简易整型转字符串
func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var buf [20]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// boolStr 将布尔值输出为 true/false
func boolStr(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// writeRows 将一批记录写入 CSV writer 并刷新缓冲，返回遇到的写入错误。
func writeRows(w *csv.Writer, records [][]string) (int64, error) {
	if len(records) == 0 {
		return 0, nil
	}
	if err := w.WriteAll(records); err != nil {
		return 0, err
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return 0, err
	}
	return int64(len(records)), nil
}

// writeHeader 写出 CSV 表头行并刷新
func writeHeader(w *csv.Writer, header []string) error {
	if err := w.Write(header); err != nil {
		return err
	}
	w.Flush()
	return w.Error()
}
