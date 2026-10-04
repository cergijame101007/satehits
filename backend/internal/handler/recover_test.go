package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// docs/monitoring.md §4.3 の 4 ケース
func TestRecover(t *testing.T) {
	t.Run("returns 500 INTERNAL_ERROR JSON and logs the stack when the handler panics", func(t *testing.T) {
		logBuf := captureLog(t)
		next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			panic("boom")
		})
		rr := httptest.NewRecorder()

		Recover(next).ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/api/v1/schedules", nil))

		if rr.Code != http.StatusInternalServerError {
			t.Fatalf("status: got %d, want %d", rr.Code, http.StatusInternalServerError)
		}
		var body ErrorResponse
		if err := json.Unmarshal(rr.Body.Bytes(), &body); err != nil {
			t.Fatalf("decode body: %v: %s", err, rr.Body.String())
		}
		if body.Error.Code != InternalErrorCode {
			t.Errorf("error.code: got %q, want %q", body.Error.Code, InternalErrorCode)
		}
		logText := logBuf.String()
		for _, want := range []string{"panic recovered", "panic=boom", "path=/api/v1/schedules", "stack_trace="} {
			if !strings.Contains(logText, want) {
				t.Errorf("log should contain %q: %s", want, logText)
			}
		}
	})

	t.Run("re-panics http.ErrAbortHandler", func(t *testing.T) {
		next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			panic(http.ErrAbortHandler)
		})
		rr := httptest.NewRecorder()

		defer func() {
			p := recover()
			err, ok := p.(error)
			if !ok || !errors.Is(err, http.ErrAbortHandler) {
				t.Fatalf("recovered: got %v, want http.ErrAbortHandler", p)
			}
			if rr.Body.Len() != 0 {
				t.Errorf("body should be empty: %s", rr.Body.String())
			}
		}()
		Recover(next).ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))
		t.Fatal("Recover should re-panic http.ErrAbortHandler")
	})

	t.Run("passes through when the handler does not panic", func(t *testing.T) {
		next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("X-Test", "1")
			w.WriteHeader(http.StatusCreated)
			if _, err := w.Write([]byte("ok")); err != nil {
				t.Errorf("write: %v", err)
			}
		})
		rr := httptest.NewRecorder()

		Recover(next).ServeHTTP(rr, httptest.NewRequest(http.MethodPost, "/", nil))

		if rr.Code != http.StatusCreated {
			t.Errorf("status: got %d, want %d", rr.Code, http.StatusCreated)
		}
		if got := rr.Body.String(); got != "ok" {
			t.Errorf("body: got %q, want %q", got, "ok")
		}
		if got := rr.Header().Get("X-Test"); got != "1" {
			t.Errorf("X-Test header: got %q, want %q", got, "1")
		}
	})

	t.Run("does not write a second response when the panic happens after the header was sent", func(t *testing.T) {
		captureLog(t)
		next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.WriteHeader(http.StatusAccepted)
			panic("after header")
		})
		rr := httptest.NewRecorder()

		Recover(next).ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/", nil))

		if rr.Code != http.StatusAccepted {
			t.Errorf("status: got %d, want %d", rr.Code, http.StatusAccepted)
		}
		if rr.Body.Len() != 0 {
			t.Errorf("body should be empty: %s", rr.Body.String())
		}
	})
}
