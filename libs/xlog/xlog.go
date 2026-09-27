package xlog

import (
	"fmt"
	"io"
	"log"
	"os"
	"runtime"
	"strings"
)

type LogLevel int

const (
	DEBUG LogLevel = iota // Value: 0 (Most verbose)
	INFO                  // Value: 1
	WARN                  // Value: 2
	ERROR                 // Value: 3
	FATAL                 // Value: 4 (Least verbose, most critical)
)

// Current log level - defaults to INFO
var currentLogLevel = initLogLevel()

func initLogLevel() LogLevel {
	levelStr := strings.ToUpper(os.Getenv("LOG_LEVEL"))
	switch levelStr {
	case "DEBUG":
		return DEBUG
	case "INFO":
		return INFO
	case "WARN":
		return WARN
	case "ERROR":
		return ERROR
	case "FATAL":
		return FATAL
	default:
		return INFO
	}
}

func SetLogLevel(level LogLevel) {
	currentLogLevel = level
}

func GetLogLevel() LogLevel {
	return currentLogLevel
}

type Logger struct {
	infoLogger  *log.Logger
	errorLogger *log.Logger
	debugLogger *log.Logger
	fatalLogger *log.Logger
	warnLogger  *log.Logger
}

var logger *Logger = newLogger()

func newLogger() *Logger {
	infoHandle := io.Writer(os.Stdout)
	errorHandle := io.Writer(os.Stderr)
	debugHandle := io.Writer(os.Stdout)
	fatalHandle := io.Writer(os.Stderr)
	warnHandle := io.Writer(os.Stdout)

	infoLogger := log.New(infoHandle, "INFO: ", log.Ldate|log.Ltime)
	errorLogger := log.New(errorHandle, "ERROR: ", log.Ldate|log.Ltime)
	debugLogger := log.New(debugHandle, "DEBUG: ", log.Ldate|log.Ltime)
	fatalLogger := log.New(fatalHandle, "FATAL: ", log.Ldate|log.Ltime)
	warnLogger := log.New(warnHandle, "WARN: ", log.Ldate|log.Ltime)

	return &Logger{
		infoLogger:  infoLogger,
		errorLogger: errorLogger,
		debugLogger: debugLogger,
		fatalLogger: fatalLogger,
		warnLogger:  warnLogger,
	}
}

func (l *Logger) infof(format string, v ...interface{}) {
	if currentLogLevel <= INFO {
		l.infoLogger.Printf(format+"\n", v...)
	}
}

func (l *Logger) errorf(format string, v ...interface{}) {
	if currentLogLevel <= ERROR {
		l.errorLogger.Printf(format+"\n", v...)
	}
}

func (l *Logger) debugf(format string, v ...interface{}) {
	if currentLogLevel <= DEBUG {
		l.debugLogger.Printf(format+"\n", v...)
	}
}

func (l *Logger) fatalf(format string, v ...interface{}) {
	l.fatalLogger.Fatalf(format+"\n", v...)
}

func (l *Logger) warnf(format string, v ...interface{}) {
	if currentLogLevel <= WARN {
		l.warnLogger.Printf(format+"\n", v...)
	}
}

// This will add entire file path
func getCallerFileNameAndLine(level ...int) string {
	skip := 2
	if len(level) > 0 {
		skip = level[0]
	}

	_, file, line, _ := runtime.Caller(skip)
	return fmt.Sprint(file, ":", line, " | ")
}

func Infof(format string, v ...interface{}) {
	logger.infof(getCallerFileNameAndLine()+format, v...)
}

func Errorf(format string, v ...interface{}) {
	logger.errorf(getCallerFileNameAndLine()+format, v...)
}

func Debugf(format string, v ...interface{}) {
	logger.debugf(getCallerFileNameAndLine()+format, v...)
}

func Fatalf(format string, v ...interface{}) {
	logger.fatalf(getCallerFileNameAndLine()+format, v...)
}

func Warnf(format string, v ...interface{}) {
	logger.warnf(getCallerFileNameAndLine()+format, v...)
}

func ErrorfPrev(format string, v ...interface{}) {
	logger.errorf(getCallerFileNameAndLine(3)+format, v...)
}
