package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// DBTX は *sql.DB と *sql.Tx の共通部分。標準ライブラリに共通型が無いため自前で定義する。
type DBTX interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// txKeyType は context のキー衝突を防ぐための非公開キー型。
type txKeyType struct{}

var txKey = txKeyType{}

func withTx(ctx context.Context, tx *sql.Tx) context.Context {
	return context.WithValue(ctx, txKey, tx)
}

func getTx(ctx context.Context) (*sql.Tx, bool) {
	tx, ok := ctx.Value(txKey).(*sql.Tx)
	return tx, ok
}

// baseRepository は各 repository が埋め込む共通基盤。
type baseRepository struct {
	db *sql.DB
}

// getDB は ctx に tx があればそれを、無ければ db を返す。
func (b *baseRepository) getDB(ctx context.Context) DBTX {
	tx, ok := getTx(ctx)
	if ok {
		return tx
	}
	return b.db
}

// TxManager は application.TxManager の実装。トランザクションの機構のみを提供する。
type TxManager struct {
	db *sql.DB
}

// NewTxManager は TxManager を生成する。
func NewTxManager(db *sql.DB) *TxManager {
	return &TxManager{db: db}
}

// DoInTx は tx を開始して fn を実行し、fn が error を返せば Rollback、nil なら Commit する。
// defer から戻り値を確定させるため名前付き戻り値にしている。
func (m *TxManager) DoInTx(ctx context.Context, fn func(ctx context.Context) error) (err error) {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
		if err != nil {
			if rbErr := tx.Rollback(); rbErr != nil {
				err = errors.Join(err, rbErr)
			}
			return
		}
		if cmErr := tx.Commit(); cmErr != nil {
			err = fmt.Errorf("commit tx: %w", cmErr)
		}
	}()
	return fn(withTx(ctx, tx))
}
