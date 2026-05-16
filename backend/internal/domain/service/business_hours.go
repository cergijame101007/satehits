package service

import "github.com/cergijame101007/satehits/internal/datetime"

// 営業時刻のデフォルト（docs/domain_knowledge.md §6 / table_design.md）
// normal・special_menu は通常営業、morning は朝営業。event はイベント都合、closed は休業のためデフォルトなし
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
func ApplyDefaultBusinessHours(scheduleType string, open, lastOrder, close datetime.Time) (datetime.Time, datetime.Time, datetime.Time) {
	defOpen, defLast, defClose, ok := defaultBusinessHoursForType(scheduleType)
	if !ok {
		return open, lastOrder, close
	}
	if open.IsZero() {
		open = defOpen
	}
	if lastOrder.IsZero() {
		lastOrder = defLast
	}
	if close.IsZero() {
		close = defClose
	}
	return open, lastOrder, close
}

func defaultBusinessHoursForType(scheduleType string) (open, lastOrder, close datetime.Time, ok bool) {
	switch scheduleType {
	case "normal", "special_menu":
		return defaultNormalOpenTime, defaultNormalLastOrderTime, defaultNormalCloseTime, true
	case "morning":
		return defaultMorningOpenTime, defaultMorningLastOrderTime, defaultMorningCloseTime, true
	default:
		// event: イベントによる / closed: 休業
		return datetime.Time{}, datetime.Time{}, datetime.Time{}, false
	}
}
