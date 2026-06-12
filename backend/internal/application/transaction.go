package application

import "context"

// TxManager はユースケースがトランザクション境界を宣言するための抽象
type TxManager interface {
	DoInTx(ctx context.Context, fn func(ctx context.Context) error) error
}
