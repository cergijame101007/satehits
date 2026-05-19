package datetime

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// DateFormat は日付の入出力形式（RFC 3339 の日付部分）。
const DateFormat = "2006-01-02"

// Date は日付のみを表す（時刻は常に 00:00:00 UTC）。
type Date struct {
	time.Time
}

// NewDate は年月日から Date を返す（UTC の日付開始時刻に正規化する）。
func NewDate(year int, month time.Month, day int) Date {
	return Date{Time: time.Date(year, month, day, 0, 0, 0, 0, time.UTC)}
}

// ParseDate は "YYYY-MM-DD" 形式の文字列を Date にパースする。
func ParseDate(s string) (Date, error) {
	t, err := time.ParseInLocation(DateFormat, s, time.UTC)
	if err != nil {
		return Date{}, fmt.Errorf("parse date %q: %w", s, err)
	}
	return Date{Time: t}, nil
}

// MustParseDate は ParseDate の panic 版（テスト専用）。
func MustParseDate(s string) Date {
	d, err := ParseDate(s)
	if err != nil {
		panic(err)
	}
	return d
}

// Weekday は暦日の曜日（UTC 午前0時基準。店舗カレンダー上の「その日」と一致する）。
func (d Date) Weekday() time.Weekday {
	if d.IsZero() {
		return time.Sunday
	}
	u := d.UTC()
	return time.Date(u.Year(), u.Month(), u.Day(), 0, 0, 0, 0, time.UTC).Weekday()
}

// AddDays は n 日後の日付を返す（負数なら過去）。
func (d Date) AddDays(n int) Date {
	if d.IsZero() {
		return Date{}
	}
	return Date{Time: d.UTC().AddDate(0, 0, n)}
}

// String は "YYYY-MM-DD" を返す（ゼロ値は空文字）。
func (d Date) String() string {
	if d.IsZero() {
		return ""
	}
	return d.UTC().Format(DateFormat)
}

// MarshalJSON は "YYYY-MM-DD" 形式の JSON 文字列にする。
func (d Date) MarshalJSON() ([]byte, error) {
	if d.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(d.String())
}

// UnmarshalJSON は "YYYY-MM-DD" または null を受け取る。
func (d *Date) UnmarshalJSON(b []byte) error {
	s := strings.TrimSpace(string(b))
	if s == "" || s == "null" {
		d.Time = time.Time{}
		return nil
	}
	s = strings.Trim(s, `"`)
	if s == "" {
		d.Time = time.Time{}
		return nil
	}
	parsed, err := ParseDate(s)
	if err != nil {
		return err
	}
	d.Time = parsed.Time
	return nil
}

// Scan は database/sql からの値を Date に読み込む。
func (d *Date) Scan(value any) error {
	if value == nil {
		d.Time = time.Time{}
		return nil
	}
	switch v := value.(type) {
	case time.Time:
		if v.IsZero() {
			d.Time = time.Time{}
			return nil
		}
		// UTC変換はせずに「年月日」のみを取得
		d.Time = time.Date(v.Year(), v.Month(), v.Day(), 0, 0, 0, 0, time.UTC)
		return nil
	case []byte:
		return d.Scan(string(v))
	case string:
		if v == "" {
			d.Time = time.Time{}
			return nil
		}
		parsed, err := ParseDate(v)
		if err != nil {
			return err
		}
		d.Time = parsed.Time
		return nil
	default:
		return fmt.Errorf("cannot scan %T into Date", value)
	}
}

// Value は DB ドライバ向けの値を返す（DATE 相当の文字列）。
func (d Date) Value() (driver.Value, error) {
	if d.IsZero() {
		return nil, nil
	}
	return d.String(), nil
}
