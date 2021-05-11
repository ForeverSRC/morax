package logger

import (
	cl "github.com/ForeverSRC/morax/config/logger"
	"log"
	"os"
)

type Logger struct {
	level int
}

const (
	debugLevel = iota
	infoLevel
	warnLevel
	errorLevel
)

var (
	debug *log.Logger
	info  *log.Logger
	warn  *log.Logger
	err   *log.Logger
)

var levelMap = map[string]int{
	"debug": debugLevel,
	"info":  infoLevel,
	"warn":  warnLevel,
	"error": errorLevel,
}

func init() {
	flags := log.Ldate | log.Ltime | log.Lmsgprefix
	debug = log.New(os.Stdout, "[DEBUG]", flags)
	info = log.New(os.Stdout, "[INFO]", flags)
	warn = log.New(os.Stdout, "[WARN]", flags)
	err = log.New(os.Stderr, "[ERROR]", flags)
}

var logger *Logger

func NewLogger(cf *cl.LoggerConfig) {
	lev, ok := levelMap[cf.Level]
	if !ok {
		lev = errorLevel
	}

	logger = &Logger{level: lev}
}

func (l *Logger) NewLogger(level string) {
	lev, ok := levelMap[level]
	if !ok {
		lev = errorLevel
	}

	l.level = lev
}

func Debug(format string, params ...interface{}) {
	if logger.level <= debugLevel {
		debug.Printf(format+"\n", params...)
	}
}

func (l *Logger) Debug(format string, params ...interface{}) {
	if l.level <= debugLevel {
		debug.Printf(format+"\n", params...)
	}
}

func Info(format string, params ...interface{}) {
	if logger.level <= infoLevel {
		info.Printf(format+"\n", params...)
	}

}

func (l *Logger) Info(format string, params ...interface{}) {
	if l.level <= infoLevel {
		info.Printf(format+"\n", params...)
	}

}

func Warn(format string, params ...interface{}) {
	if logger.level <= warnLevel {
		warn.Printf(format+"\n", params...)
	}

}

func (l *Logger) Warn(format string, params ...interface{}) {
	if l.level <= warnLevel {
		warn.Printf(format+"\n", params...)
	}

}

func Error(format string, params ...interface{}) {
	if logger.level <= errorLevel {
		err.Printf(format+"\n", params...)
	}

}

func (l *Logger) Error(format string, params ...interface{}) {
	if l.level <= errorLevel {
		err.Printf(format+"\n", params...)
	}

}

func Fatal(params ...interface{}) {
	log.Fatal(params...)
}

func (l *Logger) Fatal(params ...interface{}) {
	log.Fatal(params...)
}
