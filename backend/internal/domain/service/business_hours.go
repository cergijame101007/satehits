package service

import (
	"time"

	"github.com/cergijame101007/satehits/internal/datetime"
	"github.com/cergijame101007/satehits/internal/domain"
)

// 営業時刻のデフォルト（docs/domain_knowledge.md §6 / table_design.md）
// normal・special_menu は通常営業、morning は朝営業
// event は未指定時に暦日（祝日・曜日）で通常/朝を選んで補完（StoreCalendar.ApplyEventDefaultBusinessHours）。closed は休業のためデフォルトなし
var (
	defaultNormalOpenTime      = datetime.MustParseTime("11:30")
	defaultNormalLastOrderTime = datetime.MustParseTime("13:30")
	defaultNormalCloseTime     = datetime.MustParseTime("15:00")

	defaultMorningOpenTime      = datetime.MustParseTime("08:30")
	defaultMorningLastOrderTime = datetime.MustParseTime("13:30")
	defaultMorningCloseTime     = datetime.MustParseTime("15:00")
)

// ApplyDefaultBusinessHours は未指定（ゼロ値）の時刻に schedule_type に応じた店舗デフォルトを当てる
// 各フィールドは独立に補完する（一部だけ指定された場合は残りだけデフォルト）
func ApplyDefaultBusinessHours(scheduleType string, openTime, lastOrder, closeTime datetime.Time) (datetime.Time, datetime.Time, datetime.Time) {
	defOpen, defLast, defClose, ok := defaultBusinessHoursForType(scheduleType)
	if !ok {
		return openTime, lastOrder, closeTime
	}
	if openTime.IsZero() {
		openTime = defOpen
	}
	if lastOrder.IsZero() {
		lastOrder = defLast
	}
	if closeTime.IsZero() {
		closeTime = defClose
	}
	return openTime, lastOrder, closeTime
}

func defaultBusinessHoursForType(scheduleType string) (openTime, lastOrder, closeTime datetime.Time, ok bool) {
	switch scheduleType {
	case domain.ScheduleTypeNormal, domain.ScheduleTypeSpecialMenu:
		return defaultNormalOpenTime, defaultNormalLastOrderTime, defaultNormalCloseTime, true
	case domain.ScheduleTypeMorning:
		return defaultMorningOpenTime, defaultMorningLastOrderTime, defaultMorningCloseTime, true
	default:
		// event: StoreCalendar.ApplyEventDefaultBusinessHours を使用 / closed: 休業（時刻は保存しない）
		return datetime.Time{}, datetime.Time{}, datetime.Time{}, false
	}
}

// bookingOpenTime は予約受付開始時刻を返す（店舗開店時刻と異なる場合がある）。
func bookingOpenTime(sch domain.Schedule, storeOpen datetime.Time) datetime.Time {
	switch sch.ScheduleType {
	case domain.ScheduleTypeMorning:
		return defaultNormalOpenTime
	case domain.ScheduleTypeEvent:
		if isSundayDate(sch.Date) && timeToMinutes(storeOpen) < timeToMinutes(defaultNormalOpenTime) {
			return defaultNormalOpenTime
		}
		return storeOpen
	default:
		return storeOpen
	}
}

func isSundayDate(d datetime.Date) bool {
	if d.IsZero() {
		return false
	}
	return d.Weekday() == time.Sunday
}

func timeToMinutes(t datetime.Time) int {
	u := t.UTC()
	return u.Hour()*60 + u.Minute()
}
