// Package logging は log/slog を Cloud Logging が解釈できる JSON で出力するための設定を提供する。
// 設計は docs/monitoring.md §4.1（ADR-012）。
package logging

import (
	"io"
	"log/slog"
	"os"
)

// Cloud Logging が特別扱いする JSON キー
const (
	severityKey       = "severity"
	messageKey        = "message"
	sourceLocationKey = "logging.googleapis.com/sourceLocation"
)

// Cloud Logging の severity 列挙名。slog の WARN だけ名前が違う
const severityWarning = "WARNING"

const environmentDevelopment = "development"

// Setup は ENVIRONMENT に応じた slog.Logger を生成し slog.SetDefault する。
// development は人が読む TextHandler、それ以外は Cloud Logging 互換の JSON。
// SetDefault により log.Printf の出力も以後この logger へ INFO で流れるため、
// アラートに乗せたい行は slog.Error / slog.Warn で明示的に出す（docs/monitoring.md §8）
func Setup(environment string) *slog.Logger {
	var handler slog.Handler
	if environment == environmentDevelopment {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	} else {
		handler = NewCloudLoggingHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	}
	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger
}

// NewCloudLoggingHandler は w に Cloud Logging 互換の JSON を書く slog.Handler を返す。
// level → severity（WARN は WARNING）、msg → message、source → logging.googleapis.com/sourceLocation。
// opts の AddSource と ReplaceAttr は上書きする
func NewCloudLoggingHandler(w io.Writer, opts *slog.HandlerOptions) slog.Handler {
	o := slog.HandlerOptions{}
	if opts != nil {
		o = *opts
	}
	o.AddSource = true
	o.ReplaceAttr = replaceAttr
	return slog.NewJSONHandler(w, &o)
}

// replaceAttr はトップレベルの組み込み属性だけを Cloud Logging のキーへ置き換える。
// グループ内に同名キー（level / msg / source）があっても触らない
func replaceAttr(groups []string, a slog.Attr) slog.Attr {
	if len(groups) > 0 {
		return a
	}
	switch a.Key {
	case slog.LevelKey:
		a.Key = severityKey
		if level, ok := a.Value.Any().(slog.Level); ok {
			a.Value = slog.StringValue(severityName(level))
		}
	case slog.MessageKey:
		a.Key = messageKey
	case slog.SourceKey:
		a.Key = sourceLocationKey
	}
	return a
}

func severityName(level slog.Level) string {
	if level >= slog.LevelWarn && level < slog.LevelError {
		return severityWarning
	}
	return level.String()
}
