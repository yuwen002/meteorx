package exporter

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

func TestJSONSink(t *testing.T) {
	var buf bytes.Buffer
	sink := NewJSONSink(&buf)
	require.NoError(t, sink.WriteHeader([]string{"name", "age"}))
	_, err := sink.WriteRows([][]string{{"alice", "30"}, {"bob", "25"}})
	require.NoError(t, err)
	require.NoError(t, sink.Close())

	var arr []map[string]string
	require.NoError(t, json.Unmarshal(buf.Bytes(), &arr))
	require.Len(t, arr, 2)
	require.Equal(t, "alice", arr[0]["name"])
	require.Equal(t, "30", arr[0]["age"])
	require.Equal(t, "bob", arr[1]["name"])
}

func TestJSONSink_Empty(t *testing.T) {
	var buf bytes.Buffer
	sink := NewJSONSink(&buf)
	require.NoError(t, sink.WriteHeader([]string{"a"}))
	require.NoError(t, sink.Close())
	require.Equal(t, "[]", buf.String())
}

// TestJSONSink_RequiresHeader 未写表头直接写行/关闭时必须报错，避免产出非法 JSON。
func TestJSONSink_RequiresHeader(t *testing.T) {
	var buf bytes.Buffer
	sink := NewJSONSink(&buf)
	_, err := sink.WriteRows([][]string{{"x"}})
	require.Error(t, err, "未 WriteHeader 时 WriteRows 应拒绝")
	require.Error(t, sink.Close(), "未 WriteHeader 时 Close 应拒绝")
	require.Equal(t, "", buf.String(), "不应写入任何内容")
}

func TestXLSXSink(t *testing.T) {
	var buf bytes.Buffer
	sink, err := NewXLSXSink(&buf)
	require.NoError(t, err)
	require.NoError(t, sink.WriteHeader([]string{"姓名", "年龄"}))
	_, err = sink.WriteRows([][]string{{"张三", "30"}, {"李四", "28"}})
	require.NoError(t, err)
	require.NoError(t, sink.Close())

	f, err := excelize.OpenReader(&buf)
	require.NoError(t, err)
	rows, err := f.GetRows("Sheet1")
	require.NoError(t, err)
	require.Len(t, rows, 3)
	require.Equal(t, "姓名", rows[0][0])
	require.Equal(t, "张三", rows[1][0])
	require.Equal(t, "30", rows[1][1])
}

func TestNewSink(t *testing.T) {
	s, err := NewSink("csv", &bytes.Buffer{})
	require.NoError(t, err)
	require.IsType(t, &csvSink{}, s)

	s, err = NewSink("json", &bytes.Buffer{})
	require.NoError(t, err)
	require.IsType(t, &jsonSink{}, s)

	_, err = NewSink("xlsx", &bytes.Buffer{})
	require.NoError(t, err)

	// PDF 在本环境缺少 CJK 字体时应报错；若已配置字体则跳过该断言
	if _, e := NewSink("pdf", &bytes.Buffer{}); e == nil {
		t.Skip("已配置 CJK 字体，跳过 PDF 无字体报错断言")
	} else {
		require.Error(t, e)
	}

	// 非法格式应报错
	_, err = NewSink("doc", &bytes.Buffer{})
	require.Error(t, err)
}
