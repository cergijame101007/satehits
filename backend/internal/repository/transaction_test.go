package repository

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"errors"
	"strings"
	"testing"
)

// fakeConn / fakeTx / fakeConnector / fakeDriver は database/sql の標準ドライバ契約を
// 最小実装したテスト用フェイク。実 DB を使わず DoInTx の commit/rollback 分岐を検証する。
// （testify や sqlmock などの外部依存は入れない方針: docs/coding_rule/go_testing.md）
type fakeConn struct {
	beginErr    error
	commitErr   error
	rollbackErr error

	beginCalls    int
	commitCalls   int
	rollbackCalls int
}

func (c *fakeConn) Prepare(string) (driver.Stmt, error) { return nil, errors.New("not implemented") }
func (c *fakeConn) Close() error                        { return nil }
func (c *fakeConn) Begin() (driver.Tx, error)           { return c.begin() }

func (c *fakeConn) BeginTx(context.Context, driver.TxOptions) (driver.Tx, error) {
	return c.begin()
}

func (c *fakeConn) begin() (driver.Tx, error) {
	c.beginCalls++
	if c.beginErr != nil {
		return nil, c.beginErr
	}
	return &fakeTx{conn: c}, nil
}

type fakeTx struct{ conn *fakeConn }

func (t *fakeTx) Commit() error {
	t.conn.commitCalls++
	return t.conn.commitErr
}

func (t *fakeTx) Rollback() error {
	t.conn.rollbackCalls++
	return t.conn.rollbackErr
}

type fakeConnector struct{ conn *fakeConn }

func (c *fakeConnector) Connect(context.Context) (driver.Conn, error) { return c.conn, nil }
func (c *fakeConnector) Driver() driver.Driver                        { return fakeDriver{} }

type fakeDriver struct{}

func (fakeDriver) Open(string) (driver.Conn, error) { return nil, errors.New("use connector") }

func newFakeDB(conn *fakeConn) *sql.DB {
	return sql.OpenDB(&fakeConnector{conn: conn})
}

func TestTxManager_DoInTx(t *testing.T) {
	errFn := errors.New("fn failed")
	errRollback := errors.New("rollback failed")

	tests := []struct {
		name            string
		conn            *fakeConn
		fnErr           error
		wantErrIs       []error
		wantErrContains string
		wantFnCalled    bool
		wantCommit      int
		wantRollback    int
	}{
		{
			name:         "commits when fn returns nil",
			conn:         &fakeConn{},
			fnErr:        nil,
			wantFnCalled: true,
			wantCommit:   1,
			wantRollback: 0,
		},
		{
			name:         "rolls back and returns fn error when fn fails",
			conn:         &fakeConn{},
			fnErr:        errFn,
			wantErrIs:    []error{errFn},
			wantFnCalled: true,
			wantCommit:   0,
			wantRollback: 1,
		},
		{
			name:         "joins rollback error with fn error",
			conn:         &fakeConn{rollbackErr: errRollback},
			fnErr:        errFn,
			wantErrIs:    []error{errFn, errRollback},
			wantFnCalled: true,
			wantCommit:   0,
			wantRollback: 1,
		},
		{
			name:            "returns begin error without calling fn",
			conn:            &fakeConn{beginErr: errors.New("begin failed")},
			fnErr:           nil,
			wantErrContains: "begin tx",
			wantFnCalled:    false,
			wantCommit:      0,
			wantRollback:    0,
		},
		{
			name:            "returns commit error when commit fails",
			conn:            &fakeConn{commitErr: errors.New("commit failed")},
			fnErr:           nil,
			wantErrContains: "commit tx",
			wantFnCalled:    true,
			wantCommit:      1,
			wantRollback:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := newFakeDB(tt.conn)
			defer db.Close()
			tm := NewTxManager(db)

			fnCalled := false
			err := tm.DoInTx(context.Background(), func(context.Context) error {
				fnCalled = true
				return tt.fnErr
			})

			if fnCalled != tt.wantFnCalled {
				t.Fatalf("fn called = %v, want %v", fnCalled, tt.wantFnCalled)
			}
			for _, want := range tt.wantErrIs {
				if !errors.Is(err, want) {
					t.Fatalf("err = %v, want Is(%v)", err, want)
				}
			}
			if tt.wantErrContains != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErrContains)) {
				t.Fatalf("err = %v, want contains %q", err, tt.wantErrContains)
			}
			if len(tt.wantErrIs) == 0 && tt.wantErrContains == "" && err != nil {
				t.Fatalf("err = %v, want nil", err)
			}
			if tt.conn.commitCalls != tt.wantCommit {
				t.Fatalf("commit calls = %d, want %d", tt.conn.commitCalls, tt.wantCommit)
			}
			if tt.conn.rollbackCalls != tt.wantRollback {
				t.Fatalf("rollback calls = %d, want %d", tt.conn.rollbackCalls, tt.wantRollback)
			}
		})
	}
}

func TestTxManager_DoInTx_passes_tx_bearing_context_to_fn(t *testing.T) {
	db := newFakeDB(&fakeConn{})
	defer db.Close()
	tm := NewTxManager(db)

	err := tm.DoInTx(context.Background(), func(ctx context.Context) error {
		if _, ok := getTx(ctx); !ok {
			t.Error("fn did not receive a tx-bearing context")
		}
		return nil
	})
	if err != nil {
		t.Fatalf("DoInTx: %v", err)
	}
}

func TestTxManager_DoInTx_rolls_back_and_repanics_when_fn_panics(t *testing.T) {
	conn := &fakeConn{}
	db := newFakeDB(conn)
	defer db.Close()
	tm := NewTxManager(db)

	func() {
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic to propagate, got none")
			}
		}()
		_ = tm.DoInTx(context.Background(), func(context.Context) error {
			panic("boom")
		})
	}()

	if conn.rollbackCalls != 1 {
		t.Fatalf("rollback calls = %d, want 1", conn.rollbackCalls)
	}
	if conn.commitCalls != 0 {
		t.Fatalf("commit calls = %d, want 0", conn.commitCalls)
	}
}
