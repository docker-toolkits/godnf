package dnflog

import (
	"fmt"
	"log"
	"os"
	"runtime"
)

type LogLevel int

var L *Logger

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
)

type Logger struct {
	level   LogLevel
	logger  *log.Logger
	logFile *os.File
}

func NewLogger(level LogLevel, logFilePath string) (*Logger, error) {
	var logFile *os.File
	var err error

	if logFilePath != "" {
		logFile, err = os.OpenFile(logFilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("error opening log file: %v", err)
		}
	} else {
		logFile = os.Stdout
	}

	logger := log.New(logFile, "", log.LstdFlags)
	return &Logger{
		level:   level,
		logger:  logger,
		logFile: logFile,
	}, nil
}

func (l *Logger) Close() {
	if l.logFile != os.Stdout {
		l.logFile.Close()
	}
}

func (l *Logger) logMessage(level LogLevel, format string, v ...interface{}) {
	if level >= l.level {
		prefix := ""
		switch level {
		case DEBUG:
			prefix = "DEBUG"
		case INFO:
			prefix = "INFO"
		case WARN:
			prefix = "WARN"
		case ERROR:
			prefix = "ERROR"
		}
		_, file, line, ok := runtime.Caller(2)
		if ok {
			l.logger.SetPrefix(fmt.Sprintf("[%s][%s][%d] ", prefix, file, line))
		} else {
			l.logger.SetPrefix(fmt.Sprintf("[%s] ", prefix))
		}

		if format == "" {
			l.logger.Output(2, fmt.Sprint(v...))
		} else {
			l.logger.Output(2, fmt.Sprintf(format, v...))
		}
	}
}

func (l *Logger) Debug(format string, v ...interface{}) {
	l.logMessage(DEBUG, format, v...)
}

func (l *Logger) Info(format string, v ...interface{}) {
	l.logMessage(INFO, format, v...)
}

func (l *Logger) Warn(format string, v ...interface{}) {
	l.logMessage(WARN, format, v...)
}

func (l *Logger) Error(format string, v ...interface{}) {
	l.logMessage(ERROR, format, v...)
}
