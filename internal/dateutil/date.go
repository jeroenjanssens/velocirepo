package dateutil

import "time"

const DateLayout = "2006-01-02"

func ParseDate(s string) (time.Time, error) {
	return time.Parse(DateLayout, s)
}

func FormatDate(t time.Time) string {
	return t.Format(DateLayout)
}

func LastDayOfMonth(yearMonth string) string {
	t, err := time.Parse("2006-01", yearMonth)
	if err != nil {
		return yearMonth + "-28"
	}
	return FormatDate(t.AddDate(0, 1, -1))
}
