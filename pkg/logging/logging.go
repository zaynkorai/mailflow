package logging

import (
	"log"
	"os"
)

type Logger struct {
	*log.Logger
}

var defaultLogger *Logger

func InitLogger() {
	defaultLogger = &Logger{
		Logger: log.New(os.Stdout, "MAILFLOW: ", log.Ldate|log.Ltime|log.Lshortfile),
	}
	defaultLogger.Println("Logger initialized.")
}

func Info(format string, v ...interface{}) {
	if defaultLogger == nil {
		log.Printf("Logger not initialized. Falling back to default log: "+format, v...)
		return
	}
	defaultLogger.Printf("INFO: "+format, v...)
}

func Error(format string, v ...interface{}) {
	if defaultLogger == nil {
		log.Printf("Logger not initialized. Falling back to default log: "+format, v...)
		return
	}
	defaultLogger.Printf("ERROR: "+format, v...)
}

func Fatal(format string, v ...interface{}) {
	if defaultLogger == nil {
		log.Fatalf("Logger not initialized. Falling back to default log: "+format, v...)
		return
	}
	defaultLogger.Fatalf("FATAL: "+format, v...)
}

func Debug(format string, v ...interface{}) {
	if defaultLogger == nil {
		log.Printf("Logger not initialized. Falling back to default log: "+format, v...)
		return
	}
	defaultLogger.Printf("DEBUG: "+format, v...)
}
