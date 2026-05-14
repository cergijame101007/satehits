package usecase

import (
	"context"

	"github.com/cergijame101007/satehits/internal/datetime"
	"github.com/cergijame101007/satehits/internal/domain"
)

// FieldViolation はバリデーションで蓄えるフィールドごとのエラー
// handler の ErrorDetail と同じ形の別定義
type FieldViolation struct {
	Field   string
	Message string
}

// ValidationError はユースケース入力の検証に失敗したときに返すエラー。
type ValidationError struct {
	Violations []FieldViolation
}

func (e *ValidationError) Error() string {
	return "入力内容に誤りがあります"
}

// CreateReservationCommand は顧客による Web 予約申請の入力。
type CreateReservationCommand struct {
	Name      string
	People    int
	VisitDate datetime.Date
	VisitTime datetime.Time
	Phone     string
	Email     string
	Note      string
}

// CreateReservationUseCase は予約作成（顧客向け）を実行する。
type CreateReservationUseCase struct {
	repo domain.ReservationRepository
}

// NewCreateReservationUseCase は CreateReservationUseCase を生成する。
func NewCreateReservationUseCase(repo domain.ReservationRepository) *CreateReservationUseCase {
	return &CreateReservationUseCase{repo: repo}
}

// Execute は入力検証のうえ Repository に永続化を委譲する。
func (u *CreateReservationUseCase) Execute(ctx context.Context, cmd CreateReservationCommand) error {
	var violations []FieldViolation
	if cmd.Name == "" {
		violations = append(violations, FieldViolation{Field: "name", Message: "名前は必須です"})
	}
	if cmd.People < 1 || cmd.People > 7 {
		violations = append(violations, FieldViolation{Field: "people", Message: "人数は1〜7名で指定してください"})
	}
	if cmd.VisitDate.IsZero() {
		violations = append(violations, FieldViolation{Field: "visit_date", Message: "来店日は必須です"})
	}
	if cmd.VisitTime.IsZero() {
		violations = append(violations, FieldViolation{Field: "visit_time", Message: "来店時間は必須です"})
	}
	if cmd.Phone == "" {
		violations = append(violations, FieldViolation{Field: "phone", Message: "電話番号は必須です"})
	}
	if cmd.Email == "" {
		violations = append(violations, FieldViolation{Field: "email", Message: "メールアドレスは必須です"})
	}
	if len(violations) > 0 {
		return &ValidationError{Violations: violations}
	}

	in := domain.CreateReservationInput{
		Name:      cmd.Name,
		People:    cmd.People,
		VisitDate: cmd.VisitDate,
		VisitTime: cmd.VisitTime,
		Phone:     cmd.Phone,
		Email:     cmd.Email,
		Note:      cmd.Note,
		Status:    "pending",
		Source:    "web",
	}
	return u.repo.Create(ctx, in)
}
