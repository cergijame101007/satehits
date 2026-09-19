package logging

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"testing"
)

// decodeLine は 1 行の JSON ログをトップレベルのキーごとに分解する
func decodeLine(t *testing.T, buf *bytes.Buffer) map[string]json.RawMessage {
	t.Helper()
	var entry map[string]json.RawMessage
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("log line is not JSON: %v: %s", err, buf.String())
	}
	return entry
}

func stringField(t *testing.T, entry map[string]json.RawMessage, key string) string {
	t.Helper()
	raw, ok := entry[key]
	if !ok {
		t.Fatalf("key %q is missing: %v", key, entry)
	}
	var s string
	if err := json.Unmarshal(raw, &s); err != nil {
		t.Fatalf("key %q is not a string: %s", key, raw)
	}
	return s
}

// Cloud Logging の severity 列挙名に合わせる（docs/monitoring.md §4.1）
func TestCloudLoggingHandler_severity(t *testing.T) {
	tests := []struct {
		name  string
		level slog.Level
		want  string
	}{
		{name: "maps debug to DEBUG", level: slog.LevelDebug, want: "DEBUG"},
		{name: "maps info to INFO", level: slog.LevelInfo, want: "INFO"},
		{name: "maps warn to WARNING", level: slog.LevelWarn, want: "WARNING"},
		{name: "maps error to ERROR", level: slog.LevelError, want: "ERROR"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := slog.New(NewCloudLoggingHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))

			logger.Log(t.Context(), tt.level, "hello")

			entry := decodeLine(t, &buf)
			if got := stringField(t, entry, "severity"); got != tt.want {
				t.Errorf("severity: got %q, want %q", got, tt.want)
			}
			if _, ok := entry["level"]; ok {
				t.Errorf("level key should be replaced by severity: %v", entry)
			}
		})
	}
}

func TestCloudLoggingHandler_keys(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(NewCloudLoggingHandler(&buf, nil))

	logger.Error("create reservation failed", "reservation_id", "r-1", slog.Group("req", "msg", "inner"))

	entry := decodeLine(t, &buf)

	t.Run("writes the message under message", func(t *testing.T) {
		if got := stringField(t, entry, "message"); got != "create reservation failed" {
			t.Errorf("message: got %q, want %q", got, "create reservation failed")
		}
		if _, ok := entry["msg"]; ok {
			t.Errorf("msg key should be replaced by message: %v", entry)
		}
	})

	t.Run("writes the source location under the Cloud Logging key", func(t *testing.T) {
		raw, ok := entry["logging.googleapis.com/sourceLocation"]
		if !ok {
			t.Fatalf("sourceLocation key is missing: %v", entry)
		}
		var loc struct {
			File string `json:"file"`
			Line int    `json:"line"`
		}
		if err := json.Unmarshal(raw, &loc); err != nil {
			t.Fatalf("sourceLocation is not an object: %s", raw)
		}
		if loc.File == "" || loc.Line == 0 {
			t.Errorf("sourceLocation: got %+v, want file and line", loc)
		}
	})

	t.Run("keeps custom attributes at the top level", func(t *testing.T) {
		if got := stringField(t, entry, "reservation_id"); got != "r-1" {
			t.Errorf("reservation_id: got %q, want %q", got, "r-1")
		}
	})

	t.Run("does not rename keys inside groups", func(t *testing.T) {
		var group map[string]string
		if err := json.Unmarshal(entry["req"], &group); err != nil {
			t.Fatalf("req group: %v", err)
		}
		if group["msg"] != "inner" {
			t.Errorf("req.msg: got %q, want %q", group["msg"], "inner")
		}
	})
}

func TestCloudLoggingHandler_respectsLevel(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(NewCloudLoggingHandler(&buf, &slog.HandlerOptions{Level: slog.LevelInfo}))

	logger.Debug("hidden")

	if buf.Len() != 0 {
		t.Errorf("debug line should be dropped at info level: %s", buf.String())
	}
}
