package service

import (
	"time"

	"github.com/cergijame101007/satehits/internal/datetime"
	"github.com/cergijame101007/satehits/internal/domain"
)

// NationalHolidayChecker は国民の祝日・休日の判定。本番は holiday.Embedded()（同梱の内閣府 CSV）を注入する。
// 休業日（AvailabilityResult.IsHoliday / API の is_holiday）とは別概念
type NationalHolidayChecker interface {
	IsNationalHoliday(d datetime.Date) bool
}

// StoreCalendar は店舗定例（祝日 → 曜日）に基づく営業設定・営業時刻の合成（docs/domain_knowledge.md §3・§4・§6 / docs/holidays.md）。
// 祝日判定を注入で受け取るため、テストでは同梱データに依存せず任意の祝日集合で検証できる
type StoreCalendar struct {
	holidays NationalHolidayChecker
}

// NewStoreCalendar は StoreCalendar を生成する
func NewStoreCalendar(holidays NationalHolidayChecker) *StoreCalendar {
	return &StoreCalendar{holidays: holidays}
}

// DefaultSchedule — daily_schedules 行なし日の店舗定例合成
// 祝日は曜日にかかわらず normal（日曜と重なっても朝営業なし）、それ以外は木金 closed・日曜 morning・月〜水土 normal
func (c *StoreCalendar) DefaultSchedule(d datetime.Date) domain.Schedule {
	scheduleType, capacity := c.defaultScheduleTypeAndCapacity(d)
	openTime, lastOrder, closeTime := ApplyDefaultBusinessHours(scheduleType, datetime.Time{}, datetime.Time{}, datetime.Time{})
	return domain.Schedule{
		Date:          d,
		ScheduleType:  scheduleType,
		Capacity:      capacity,
		OpenTime:      openTime,
		LastOrderTime: lastOrder,
		CloseTime:     closeTime,
	}
}

// defaultScheduleTypeAndCapacity は店舗定例のタイプ・提供数。祝日が最優先、次に曜日
func (c *StoreCalendar) defaultScheduleTypeAndCapacity(d datetime.Date) (scheduleType string, capacity int) {
	if c.holidays.IsNationalHoliday(d) {
		return domain.ScheduleTypeNormal, defaultScheduleCapacity
	}
	switch d.Weekday() {
	case time.Thursday, time.Friday:
		return domain.ScheduleTypeClosed, 0
	case time.Sunday:
		return domain.ScheduleTypeMorning, defaultScheduleCapacity
	default:
		return domain.ScheduleTypeNormal, defaultScheduleCapacity
	}
}

// ApplyEventDefaultBusinessHours は event で未指定の時刻に、その日の暦日に応じた店舗デフォルトを当てる
// 祝日でない日曜のみ朝営業、それ以外は通常営業（店内イベント想定。定例の木金休業は event 行で上書き）
// 祝日は日曜と重なっても朝営業なし（docs/domain_knowledge.md §4）
func (c *StoreCalendar) ApplyEventDefaultBusinessHours(date datetime.Date, openTime, lastOrder, closeTime datetime.Time) (datetime.Time, datetime.Time, datetime.Time) {
	return ApplyDefaultBusinessHours(c.defaultBusinessHoursTypeForEventDate(date), openTime, lastOrder, closeTime)
}

func (c *StoreCalendar) defaultBusinessHoursTypeForEventDate(date datetime.Date) string {
	if date.Weekday() == time.Sunday && !c.holidays.IsNationalHoliday(date) {
		return domain.ScheduleTypeMorning
	}
	return domain.ScheduleTypeNormal
}

// BookingWindowMinutes は有効スケジュールに基づく予約受付の開始・最終時刻（分）を返す。
// 店舗の開店時刻（ApplyDefaultBusinessHours）とは別。日曜朝営業は 8:30 開店だが予約は 11:30 から（domain_knowledge §4）。
// closed や時刻未定の種別では ok=false。
func (c *StoreCalendar) BookingWindowMinutes(sch domain.Schedule) (openMinutes, lastOrderMinutes int, ok bool) {
	if sch.ScheduleType == domain.ScheduleTypeClosed ||
		sch.ScheduleType == domain.ScheduleTypeExternalEvent {
		return 0, 0, false
	}

	var storeOpen, lastOrder datetime.Time
	switch sch.ScheduleType {
	case domain.ScheduleTypeEvent:
		storeOpen, lastOrder, _ = c.ApplyEventDefaultBusinessHours(sch.Date, sch.OpenTime, sch.LastOrderTime, sch.CloseTime)
	default:
		storeOpen, lastOrder, _ = ApplyDefaultBusinessHours(sch.ScheduleType, sch.OpenTime, sch.LastOrderTime, sch.CloseTime)
	}
	if storeOpen.IsZero() || lastOrder.IsZero() {
		return 0, 0, false
	}
	bookingOpen := bookingOpenTime(sch, storeOpen)
	return timeToMinutes(bookingOpen), timeToMinutes(lastOrder), true
}
