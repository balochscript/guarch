package log

import (
	"fmt"
	"io"
	stdlog "log"
	"os"
	"sync/atomic"
)

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

func SetOutput(w io.Writer) {
	if w == nil {
		w = os.Stderr
	}
	logger.SetOutput(w)
}

func SetLevel(l Level) {
	if l < LevelDebug {
		l = LevelDebug
	}
	if l > LevelNone {
		l = LevelNone
	}
	level.Store(int32(l))
}

func GetLevel() Level {
	return Level(level.Load())
}

func Silence() {
	level.Store(int32(LevelNone))
}

func logf(l Level, format string, v ...any) {
	if l < LevelDebug || l > LevelError {
		return
	}
	if Level(level.Load()) > l {
		return
	}
	if format == "" {
		return
	}
	tag := levelTags[l]
	msg := fmt.Sprintf(format, v...)
	_ = logger.Output(3, tag+" "+msg)
}

func Debugf(format string, v ...any) { logf(LevelDebug, format, v...) }
func Infof(format string, v ...any)  { logf(LevelInfo, format, v...) }
func Warnf(format string, v ...any)  { logf(LevelWarn, format, v...) }
func Errorf(format string, v ...any) { logf(LevelError, format, v...) }
func Printf(format string, v ...any) { logf(LevelInfo, format, v...) }
