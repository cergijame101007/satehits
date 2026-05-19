package service

import (
	"time"

	"github.com/cergijame101007/satehits/internal/datetime"
	"github.com/cergijame101007/satehits/internal/domain"
)

// 営業時刻のデフォルト（docs/domain_knowledge.md §6 / table_design.md）
// normal・special_menu は通常営業、morning は朝営業
// event は未指定時に暦日の曜日で通常/朝を選んで補完。closed は休業のためデフォルトなし
var (
	defaultNormalOpenTime      = datetime.MustParseTime("11:30")
	defaultNormalLastOrderTime = datetime.MustParseTime("14:00")
	defaultNormalCloseTime     = datetime.MustParseTime("15:00")

	defaultMorningOpenTime      = datetime.MustParseTime("08:30")
	defaultMorningLastOrderTime = datetime.MustParseTime("14:00")
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

// ApplyEventDefaultBusinessHours は event で未指定の時刻に、その日の曜日に応じた店舗デフォルトを当てる
// 土日は朝営業、それ以外は通常営業（店内イベント想定。定例の木金休業は event 行で上書き）
func ApplyEventDefaultBusinessHours(date datetime.Date, openTime, lastOrder, closeTime datetime.Time) (datetime.Time, datetime.Time, datetime.Time) {
	return ApplyDefaultBusinessHours(defaultBusinessHoursTypeForEventDate(date), openTime, lastOrder, closeTime)
}

func defaultBusinessHoursTypeForEventDate(date datetime.Date) string {
	switch date.Weekday() {
	case time.Saturday, time.Sunday:
		return domain.ScheduleTypeMorning
	default:
		return domain.ScheduleTypeNormal
	}
}

func defaultBusinessHoursForType(scheduleType string) (openTime, lastOrder, closeTime datetime.Time, ok bool) {
	switch scheduleType {
	case domain.ScheduleTypeNormal, domain.ScheduleTypeSpecialMenu:
		return defaultNormalOpenTime, defaultNormalLastOrderTime, defaultNormalCloseTime, true
	case domain.ScheduleTypeMorning:
		return defaultMorningOpenTime, defaultMorningLastOrderTime, defaultMorningCloseTime, true
	default:
		// event: ApplyEventDefaultBusinessHours を使用 / closed: 休業（時刻は保存しない）
		return datetime.Time{}, datetime.Time{}, datetime.Time{}, false
	}
}
