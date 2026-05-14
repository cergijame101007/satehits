package usecase

import (
	"fmt"
	"log"
	"net/mail"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/cergijame101007/satehits/internal/datetime"
)

// datetime.Date は暦日を UTC 午前0時で保持するだけでタイムゾーン付きの「来店日」ではない
// 「翌日〜14日先」は店のカレンダー基準のため time.Now を Asia/Tokyo の日付に正規化して比較する
var storeLocation = func() *time.Location {
	loc, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		return time.FixedZone("JST", 9*3600)
	}
	return loc
}()

const (
	minReservationPeople = 1
	maxReservationPeople = 7
	maxNameRunes         = 100
	maxNoteRunes         = 500
	// RFC 5321 に基づくメールアドレスの最大長（MaxBytesReader で全体を制限済みだが、フィールド単位の早期拒否に使用）
	maxEmailOctets     = 254
	minBookingLeadDays = 1
	maxBookingLeadDays = 14

	phoneFieldMinRunes  = 10
	phoneFieldMaxRunes  = 30
	phoneDigitsMinCount = 10
	phoneDigitsMaxCount = 15

	// 来店受付の境界（店舗ローカルタイム基準・分単位）
	// "BookingCutoff" は予約として受け付ける最終時刻
	weekdayBookingCutoffMinutes = 14 * 60    // 14:00
	weekendBookingCutoffMinutes = 14 * 60    // 14:00
	weekdayBookingOpenMinutes   = 11*60 + 30 // 11:30
	weekendBookingOpenMinutes   = 8*60 + 30  // 08:30
)

// 電話番号フィールドに許す文字: 数字・+ () - 半角スペース・およびフォームでよく使われる全角スペース（U+3000 IDEOGRAPHIC SPACE）
// 長さは rune 数で validPhone が検証する。Go の regexp の {n,m} は UTF-8 上の解釈でも誤解を招きやすいため、
// ここでは許容文字のみを判定し、\s（タブ・改行等）は含めない。
var phonePattern = regexp.MustCompile(`^[0-9+()\- 　]+$`)

func validateCreateReservation(cmd CreateReservationCommand, now time.Time) []FieldViolation {
	var violations []FieldViolation

	name := strings.TrimSpace(cmd.Name)
	nameRunes := utf8.RuneCountInString(name)
	if nameRunes == 0 {
		violations = append(violations, FieldViolation{Field: "name", Message: "名前は必須です"})
	} else if nameRunes > maxNameRunes {
		violations = append(violations, FieldViolation{Field: "name", Message: fmt.Sprintf("名前は1〜%d文字で入力してください", maxNameRunes)})
	}

	if cmd.People < minReservationPeople || cmd.People > maxReservationPeople {
		violations = append(violations, FieldViolation{Field: "people", Message: fmt.Sprintf("人数は%d〜%d名で指定してください", minReservationPeople, maxReservationPeople)})
	}

	var visitDateViolations []FieldViolation
	if cmd.VisitDate.IsZero() {
		violations = append(violations, FieldViolation{Field: "visit_date", Message: "来店日は必須です"})
	} else {
		visitDateViolations = validateVisitDate(cmd.VisitDate, now)
		violations = append(violations, visitDateViolations...)
	}

	if cmd.VisitTime.IsZero() {
		violations = append(violations, FieldViolation{Field: "visit_time", Message: "来店時間は必須です"})
	} else if !cmd.VisitDate.IsZero() && len(visitDateViolations) == 0 {
		violations = append(violations, validateVisitTime(cmd.VisitDate, cmd.VisitTime)...)
	}

	phone := strings.TrimSpace(cmd.Phone)
	if phone == "" {
		violations = append(violations, FieldViolation{Field: "phone", Message: "電話番号は必須です"})
	} else if !validPhone(phone) {
		violations = append(violations, FieldViolation{Field: "phone", Message: "電話番号の形式が正しくありません"})
	}

	email := strings.TrimSpace(cmd.Email)
	if email == "" {
		violations = append(violations, FieldViolation{Field: "email", Message: "メールアドレスは必須です"})
	} else if !validEmail(email) {
		violations = append(violations, FieldViolation{Field: "email", Message: "メールアドレスの形式が正しくありません"})
	}

	if utf8.RuneCountInString(cmd.Note) > maxNoteRunes {
		violations = append(violations, FieldViolation{Field: "note", Message: fmt.Sprintf("備考は%d文字以内で入力してください", maxNoteRunes)})
	}

	return violations
}

func validateVisitDate(visit datetime.Date, now time.Time) []FieldViolation {
	var violations []FieldViolation
	visitStart := visitStartInStoreCalendar(visit)

	nowDay := now.In(storeLocation)
	today := time.Date(nowDay.Year(), nowDay.Month(), nowDay.Day(), 0, 0, 0, 0, storeLocation)
	minBookable := today.AddDate(0, 0, minBookingLeadDays)
	maxBookable := today.AddDate(0, 0, maxBookingLeadDays)

	if visitStart.Before(minBookable) || visitStart.After(maxBookable) {
		violations = append(violations, FieldViolation{Field: "visit_date", Message: "来店日は翌日から14日以内で指定してください"})
	}

	// ドメイン既定の定休（木・金）。daily_schedules で上書きされる前提の暫定ルール
	wd := civilWeekday(visit)
	if wd == time.Thursday || wd == time.Friday {
		violations = append(violations, FieldViolation{Field: "visit_date", Message: "この日は予約できません"})
	}

	return violations
}

// visitStartInStoreCalendar は Date の暦日を店舗ロケーションの同日 0:00 に載せた瞬間（日付の大小比較用）
func visitStartInStoreCalendar(d datetime.Date) time.Time {
	if d.IsZero() {
		return time.Time{}
	}
	u := d.UTC()
	return time.Date(u.Year(), u.Month(), u.Day(), 0, 0, 0, 0, storeLocation)
}

// civilWeekday は datetime.Date が表す暦日の曜日（UTC 午前0時での Weekday と一致する）
func civilWeekday(d datetime.Date) time.Weekday {
	if d.IsZero() {
		return time.Sunday
	}
	u := d.UTC()
	return time.Date(u.Year(), u.Month(), u.Day(), 0, 0, 0, 0, time.UTC).Weekday()
}

// validateVisitTime は api_design の曜日別営業時間内かどうか（祝日・朝枠・daily_schedules は未考慮）
// visit_date 側で弾かれた日（木金等）は呼び出さないこと
// 想定外の曜日が来た場合はログを残しつつユーザーには「予約できない」と返して安全側に倒す
func validateVisitTime(visit datetime.Date, visitTime datetime.Time) []FieldViolation {
	if visitTime.IsZero() || visit.IsZero() {
		return nil
	}
	wd := civilWeekday(visit)
	mins := visitMinutes(visitTime)

	// TODO: 日本の祝日は土日祝と同じ来店時間帯で検証する（現状は曜日のみ）
	// TODO: domain_knowledge の「朝営業は予約不可」と api_design の土日 8:30 開始の整合を daily_schedules 側で扱う

	switch wd {
	case time.Monday, time.Tuesday, time.Wednesday:
		if mins < weekdayBookingOpenMinutes || mins > weekdayBookingCutoffMinutes {
			return []FieldViolation{{Field: "visit_time", Message: "来店時間が営業時間外です"}}
		}
	case time.Saturday, time.Sunday:
		if mins < weekendBookingOpenMinutes || mins > weekendBookingCutoffMinutes {
			return []FieldViolation{{Field: "visit_time", Message: "来店時間が営業時間外です"}}
		}
	default:
		// 本来 visit_date 側で弾かれているため到達しない。バグの兆候としてログを残し、
		// ユーザーには「予約できない」と返して処理を続行する（panic させず安全側に倒す）
		log.Printf("reservation: validateVisitTime invariant broken weekday=%v visit=%s", wd, visit.String())
		return []FieldViolation{{Field: "visit_time", Message: "この日は予約できません"}}
	}
	return nil
}

func visitMinutes(t datetime.Time) int {
	u := t.UTC()
	return u.Hour()*60 + u.Minute()
}

func validPhone(s string) bool {
	n := utf8.RuneCountInString(s)
	if n < phoneFieldMinRunes || n > phoneFieldMaxRunes {
		return false
	}
	if !phonePattern.MatchString(s) {
		return false
	}
	digits := 0
	for _, r := range s {
		if r >= '0' && r <= '9' {
			digits++
		}
	}
	return digits >= phoneDigitsMinCount && digits <= phoneDigitsMaxCount
}

// validEmail は API フィールド用の単純なメール文字列のみ許可する
// 角括弧付きの RFC 5322 形式（例: "田中 <tanaka@example.com>"）や Display Name 付きは拒否する
func validEmail(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	if len(s) > maxEmailOctets {
		return false
	}
	if strings.ContainsAny(s, "<>") {
		return false
	}
	addr, err := mail.ParseAddress(s)
	if err != nil {
		return false
	}
	if addr.Name != "" {
		return false
	}
	// 1 フィールドに収まったアドレスのみ
	return addr.Address == s
}
