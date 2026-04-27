package log

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"
)

type Level string

const (
	LevelDebug Level = "debug"
	LevelInfo  Level = "info"
	LevelWarn  Level = "warn"
	LevelError Level = "error"
)

type Logger struct {
	mu     sync.Mutex
	out    io.Writer
	errOut io.Writer
	level  Level
	format string
}

func New(out io.Writer, errOut io.Writer, level, format string) *Logger {
	if level == "" {
		level = "info"
	}
	if format == "" {
		format = "text"
	}
	return &Logger{out: out, errOut: errOut, level: Level(strings.ToLower(level)), format: strings.ToLower(format)}
}

func (l *Logger) Infof(format string, args ...any)  { l.write(l.out, LevelInfo, format, args...) }
func (l *Logger) Warnf(format string, args ...any)  { l.write(l.errOut, LevelWarn, format, args...) }
func (l *Logger) Errorf(format string, args ...any) { l.write(l.errOut, LevelError, format, args...) }
func (l *Logger) Debugf(format string, args ...any) { l.write(l.out, LevelDebug, format, args...) }

func (l *Logger) write(w io.Writer, level Level, format string, args ...any) {
	if l == nil || w == nil {
		return
	}
	if !l.enabled(level) {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	msg := fmt.Sprintf(format, args...)
	switch l.format {
	case "json":
		payload, _ := json.Marshal(map[string]any{
			"time":  time.Now().UTC().Format(time.RFC3339),
			"level": string(level),
			"msg":   msg,
		})
		fmt.Fprintln(w, string(payload))
	default:
		fmt.Fprintf(w, "%s: %s\n", strings.ToUpper(string(level)), msg)
	}
}

func (l *Logger) enabled(level Level) bool {
	order := map[Level]int{LevelDebug: 0, LevelInfo: 1, LevelWarn: 2, LevelError: 3}
	return order[level] >= order[l.level]
}
