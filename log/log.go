package log

import (
	"log"
	"os"
)

var (
	debugLogger *log.Logger
	infoLogger  *log.Logger
	warnLogger  *log.Logger
	errorLogger *log.Logger
)

func init() {
	SetLoggers("traefik_s3_plugin")
}

// SetLoggers initializes the loggers with the specified prefix and log level.
// If no prefix is provided, an empty string is used.
func SetLoggers(prefix string) {
	debugLogger = log.New(os.Stdout, prefix+" DEBUG: ", log.Ldate|log.Ltime|log.Lshortfile)
	infoLogger = log.New(os.Stdout, prefix+" INFO: ", log.Ldate|log.Ltime|log.Lshortfile)
	warnLogger = log.New(os.Stderr, prefix+" WARN: ", log.Ldate|log.Ltime|log.Lshortfile)
	errorLogger = log.New(os.Stderr, prefix+" ERROR: ", log.Ldate|log.Ltime|log.Lshortfile)
}

func Debug(v ...interface{}) {
	if debugLogger != nil {
		debugLogger.Println(v...)
	}
}

func Info(v ...interface{}) {
	if infoLogger != nil {
		infoLogger.Println(v...)
	}
}

func Warn(v ...interface{}) {
	if warnLogger != nil {
		warnLogger.Println(v...)
	}
}

func Error(v ...interface{}) {
	if errorLogger != nil {
		errorLogger.Println(v...)
	}
}
