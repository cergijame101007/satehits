package datetime

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// TimeFormat は時刻の入出力レイアウト（分まで）
const TimeFormat = "15:04"

// timeRef は時刻のみ保持するための参照日付（ドメイン意味なし）
var timeRef = time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC)

// Time はその日の時刻のみ（日付・タイムゾーンなし）
type Time struct {
	time.Time
}

// NewTime は時・分からの構築（秒は常に 0）
func NewTime(hour, minute int) Time {
	return Time{Time: time.Date(timeRef.Year(), timeRef.Month(), timeRef.Day(), hour, minute, 0, 0, time.UTC)}
}

// ParseTime は "HH:MM" のパース（秒付き文字列はエラー）
func ParseTime(s string) (Time, error) {
	parsed, err := time.ParseInLocation(TimeFormat, s, time.UTC)
	if err != nil {
		return Time{}, fmt.Errorf("parse time %q: %w", s, err)
	}
	return Time{Time: time.Date(timeRef.Year(), timeRef.Month(), timeRef.Day(), parsed.Hour(), parsed.Minute(), 0, 0, time.UTC)}, nil
}

// MustParseTime は ParseTime の panic 版（テスト専用）
func MustParseTime(s string) Time {
	tm, err := ParseTime(s)
	if err != nil {
		panic(err)
	}
	return tm
}

// String は "HH:MM" 表現（ゼロ値は空文字）
func (tm Time) String() string {
	if tm.IsZero() {
		return ""
	}
	u := tm.UTC()
	return time.Date(timeRef.Year(), timeRef.Month(), timeRef.Day(), u.Hour(), u.Minute(), 0, 0, time.UTC).Format(TimeFormat)
}

// MarshalJSON は JSON への "HH:MM" 文字列化
func (tm Time) MarshalJSON() ([]byte, error) {
	if tm.IsZero() {
		return []byte("null"), nil
	}
	return json.Marshal(tm.String())
}

// UnmarshalJSON は JSON からの "HH:MM" または null
func (tm *Time) UnmarshalJSON(b []byte) error {
	s := strings.TrimSpace(string(b))
	if s == "" || s == "null" {
		tm.Time = time.Time{}
		return nil
	}
	s = strings.Trim(s, `"`)
	if s == "" {
		tm.Time = time.Time{}
		return nil
	}
	parsed, err := ParseTime(s)
	if err != nil {
		return err
	}
	tm.Time = parsed.Time
	return nil
}

// Scan は database/sql からの読み込み
// HH:MM:SS 文字列および秒付き time.Time の可能性
// ドメインは分粒度のため秒は切り捨て
func (tm *Time) Scan(value any) error {
	if value == nil {
		tm.Time = time.Time{}
		return nil
	}
	switch v := value.(type) {
	case time.Time:
		if v.IsZero() {
			tm.Time = time.Time{}
			return nil
		}
		tm.Time = time.Date(timeRef.Year(), timeRef.Month(), timeRef.Day(), v.Hour(), v.Minute(), 0, 0, time.UTC)
		return nil
	case []byte:
		return tm.Scan(string(v))
	case string:
		if v == "" {
			tm.Time = time.Time{}
			return nil
		}
		// "15:04:05" と TimeFormat の順でレイアウト試行
		layouts := []string{"15:04:05", TimeFormat}
		var lastErr error
		for _, layout := range layouts {
			parsed, err := time.ParseInLocation(layout, v, time.UTC)
			if err != nil {
				lastErr = err
				continue
			}
			tm.Time = time.Date(timeRef.Year(), timeRef.Month(), timeRef.Day(), parsed.Hour(), parsed.Minute(), 0, 0, time.UTC)
			return nil
		}
		return fmt.Errorf("parse time from string %q: %w", v, lastErr)
	default:
		return fmt.Errorf("cannot scan %T into datetime.Time", value)
	}
}

// Value は DB ドライバ向け値（TIME 相当の "HH:MM:SS" 文字列）
func (tm Time) Value() (driver.Value, error) {
	if tm.IsZero() {
		return nil, nil
	}
	u := tm.UTC()
	return time.Date(timeRef.Year(), timeRef.Month(), timeRef.Day(), u.Hour(), u.Minute(), 0, 0, time.UTC).Format("15:04:05"), nil
}
