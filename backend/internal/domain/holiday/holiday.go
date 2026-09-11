// Package holiday は同梱した内閣府「国民の祝日」CSV に基づく祝日判定を提供する。
//
// データ出典: https://www8.cao.go.jp/chosei/shukujitsu/syukujitsu.csv（Shift_JIS）
// syukujitsu.csv は UTF-8 に変換したものを同梱し、年 1 回 `make update-holidays` で更新する（docs/holidays.md）。
// リクエストのたびに外部へ取りに行かないため、同梱データに無い年は「祝日なし」として扱う。
package holiday

import (
	"bufio"
	_ "embed"
	"io"
	"strings"
	"time"

	"github.com/cergijame101007/satehits/internal/datetime"
)

//go:embed syukujitsu.csv
var embeddedCSV string

// sourceDateLayout は内閣府 CSV の日付形式（ゼロ埋めなし。例: 2026/1/1）
const sourceDateLayout = "2006/1/2"

// Set は祝日の集合（日付のみ。名称は判定に使わない）
type Set struct {
	dates    map[string]struct{}
	lastYear int
}

// Parse は内閣府 CSV 形式（1 列目が YYYY/M/D）の祝日一覧を読み込む。
// ヘッダ行・空行・日付として解釈できない行はスキップする（BOM 付きも可）。
// error は読み取り失敗時のみ返す。
func Parse(r io.Reader) (*Set, error) {
	s := &Set{dates: make(map[string]struct{})}
	sc := bufio.NewScanner(r)
	first := true
	for sc.Scan() {
		line := sc.Text()
		if first {
			line = strings.TrimPrefix(line, "\uFEFF")
			first = false
		}
		d, ok := parseLine(line)
		if !ok {
			continue
		}
		s.dates[d.String()] = struct{}{}
		if y := d.Year(); y > s.lastYear {
			s.lastYear = y
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	return s, nil
}

func parseLine(line string) (datetime.Date, bool) {
	field, _, _ := strings.Cut(line, ",")
	field = strings.TrimSpace(field)
	if field == "" {
		return datetime.Date{}, false
	}
	t, err := time.ParseInLocation(sourceDateLayout, field, time.UTC)
	if err != nil {
		return datetime.Date{}, false
	}
	return datetime.NewDate(t.Year(), t.Month(), t.Day()), true
}

// Contains は d が祝日なら true（ゼロ値・データに無い年は false）
func (s *Set) Contains(d datetime.Date) bool {
	if s == nil || d.IsZero() {
		return false
	}
	_, ok := s.dates[d.String()]
	return ok
}

// LastYear はデータに含まれる最終年（空なら 0）
func (s *Set) LastYear() int {
	if s == nil {
		return 0
	}
	return s.lastYear
}

// Len は祝日の件数
func (s *Set) Len() int {
	if s == nil {
		return 0
	}
	return len(s.dates)
}

// NeedsUpdate は同梱データの最終年が now の年以前（翌年分が未同梱）なら true。
// 内閣府 CSV の翌年分は例年 2 月頃に公開されるため、起動時の警告に使う。
func (s *Set) NeedsUpdate(now time.Time) bool {
	return s.LastYear() <= now.Year()
}

var embedded = mustParseEmbedded()

func mustParseEmbedded() *Set {
	s, err := Parse(strings.NewReader(embeddedCSV))
	if err != nil {
		panic("holiday: parse embedded syukujitsu.csv: " + err.Error())
	}
	return s
}

// IsHoliday は同梱データに基づき d が国民の祝日・休日なら true を返す
func IsHoliday(d datetime.Date) bool {
	return embedded.Contains(d)
}

// Embedded は同梱データの祝日集合（起動時の鮮度チェック等に使う）
func Embedded() *Set {
	return embedded
}
