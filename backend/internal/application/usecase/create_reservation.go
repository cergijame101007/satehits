package usecase

import (
	"context"

	"github.com/cergijame101007/satehits/internal/datetime"
	"github.com/cergijame101007/satehits/internal/domain"
)

// FieldViolation はフィールド単位のバリデーションエラー
// handler.ErrorDetail と同形の別定義（handler 非依存のため）
type FieldViolation struct {
	Field   string
	Message string
}

// ValidationError はユースケース入力の検証失敗
type ValidationError struct {
	Violations []FieldViolation
}

func (e *ValidationError) Error() string {
	return "入力内容に誤りがあります"
}

// CreateReservationCommand は顧客向け Web 予約申請の入力
type CreateReservationCommand struct {
	Name      string
	People    int
	VisitDate datetime.Date
	VisitTime datetime.Time
	Phone     string
	Email     string
	Note      string
}

// CreateReservationUseCase は顧客向け予約作成
type CreateReservationUseCase struct {
	repo domain.ReservationRepository
}

// NewCreateReservationUseCase は CreateReservationUseCase の生成
func NewCreateReservationUseCase(repo domain.ReservationRepository) *CreateReservationUseCase {
	return &CreateReservationUseCase{repo: repo}
}

// Execute は入力検証および Repository への永続化
func (u *CreateReservationUseCase) Execute(ctx context.Context, cmd CreateReservationCommand) (*domain.Reservation, error) {
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
		return nil, &ValidationError{Violations: violations}
	}

	// TODO: daily_schedules と AvailabilityService 実装後にここで残席・営業可否を検証する
	// 設計どおり提供数超過は 409 CAPACITY_EXCEEDED、不可日時はバリデーションで弾く（現状は未チェック）

	// ドメイン入力への変換
	// Status/Source はサーバー側管理（クライアント非公開）
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
	res, err := u.repo.Create(ctx, in)
	if err != nil {
		return nil, err
	}
	return &res, nil
}
