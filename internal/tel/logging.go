package tel

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/lmittmann/tint"
)

type Logger struct {
	log *slog.Logger
}

func NewLogger(log *slog.Logger) Logger {
	return Logger{
		log: log,
	}
}

func (t Logger) Debug(component, msg string, args ...any) {
	t.log.Debug(fmt.Sprintf("[%s] %s", component, msg), args...)
}

func (t Logger) Info(component, msg string, args ...any) {
	t.log.Info(fmt.Sprintf("[%s] %s", component, msg), args...)
}

func (t Logger) Warn(component, msg string, args ...any) {
	t.log.Warn(fmt.Sprintf("[%s] %s", component, msg), args...)
}

func (t Logger) Error(component, msg string, args ...any) {
	t.log.Error(fmt.Sprintf("[%s] %s", component, msg), args...)
}

var Log Logger

func init() {
	Log = NewLogger(slog.New(
		tint.NewHandler(os.Stderr, &tint.Options{
			Level: slog.LevelDebug,
		}),
	))
}
