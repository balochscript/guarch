package log

import (
	"fmt"
	"io"
	stdlog "log"
	"os"
	"sync/atomic"
)

// ═══════════════════════════════════════
// ✅ L4: Logger interface for library code
//
// Library packages (transport, mux, cover, ...)
// should use this instead of standard log.Printf
//
// Benefits:
//   - Callers can silence logs: log.SetOutput(io.Discard)
//   - Callers can redirect: log.SetOutput(file)
//   - Callers can set level: log.SetLevel(LevelError)
//   - Tests can capture logs
// ═══════════════════════════════════════

type Level int32

const (
	LevelDebug Level = 0
	LevelInfo  Level = 1
	LevelWarn  Level = 2
	LevelError Level = 3
	LevelNone  Level = 4
)

var (
	logger    = stdlog.New(os.Stderr, "", stdlog.LstdFlags)
	level     atomic.Int32
	levelTags = [...]string{"[DBG]", "[INF]", "[WRN]", "[ERR]"}
)

func init() {
	level.Store(int32(LevelInfo))
}

// SetOutput — تنظیم خروجی لاگ
func SetOutput(w io.Writer) {
	logger.SetOutput(w)
}

// SetLevel — حداقل سطح لاگ
func SetLevel(l Level) {
	level.Store(int32(l))
}

// GetLevel — سطح فعلی
func GetLevel() Level {
	return Level(level.Load())
}

// Silence — خاموش کردن همه لاگ‌ها
func Silence() {
	level.Store(int32(LevelNone))
}

func logf(l Level, format string, v ...any) {
	if Level(level.Load()) > l {
		return
	}
	tag := levelTags[l]
	msg := fmt.Sprintf(format, v...)
	logger.Output(3, tag+" "+msg)
}

// Debugf — لاگ debug (مخفی در حالت عادی)
func Debugf(format string, v ...any) { logf(LevelDebug, format, v...) }

// Infof — لاگ اطلاعاتی
func Infof(format string, v ...any) { logf(LevelInfo, format, v...) }

// Warnf — هشدار
func Warnf(format string, v ...any) { logf(LevelWarn, format, v...) }

// Errorf — خطا
func Errorf(format string, v ...any) { logf(LevelError, format, v...) }

// Printf — سازگار با log.Printf (سطح Info)
func Printf(format string, v ...any) { logf(LevelInfo, format, v...) }
