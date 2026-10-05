// Package timeutil 提供基于业务时区的时间边界计算助手。
//
// 项目在启动时通过 bootstrap.applyTimezone 将全局 time.Local 强制设为业务时区
// （database.timezone，默认 Asia/Shanghai），因此 time.Now() 返回的即为业务时区时间。
// 做“今日/本周/本月”等按自然日统计时，必须使用该时区下的本地日历边界，
// 而不能使用 time.Truncate(24*time.Hour)——后者基于 UTC 绝对时间截断，
// 在非 UTC 时区会得到当地 08:00 之类的错误边界。
package timeutil

import "time"

// StartOfDay 返回 t 所在自然日的本地零点（00:00:00），保留 t 的时区。
// 用于替代 time.Now().Truncate(24*time.Hour) 这类错误的日期分桶写法。
func StartOfDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

// StartOfWeek 返回以周一为起点的、t 所在自然周的本地零点。
func StartOfWeek(t time.Time) time.Time {
	y, m, d := t.Date()
	weekday := int(t.Weekday())
	if weekday == 0 { // Sunday 归为上一周的第 7 天
		weekday = 7
	}
	return time.Date(y, m, d-(weekday-1), 0, 0, 0, 0, t.Location())
}

// StartOfMonth 返回 t 所在自然月的本地一号零点。
func StartOfMonth(t time.Time) time.Time {
	y, m, _ := t.Date()
	return time.Date(y, m, 1, 0, 0, 0, 0, t.Location())
}
