package logger

import "log"

type Logger struct{}

func New(_ string) *Logger {
	return &Logger{}
}

func (l *Logger) Info(msg string) {
	log.Printf("INFO: %s", msg)
}

func (l *Logger) Infof(format string, args ...any) {
	log.Printf("INFO: "+format, args...)
}

func (l *Logger) Errorf(format string, args ...any) {
	log.Printf("ERROR: "+format, args...)
}

func (l *Logger) Fatalf(format string, args ...any) {
	log.Fatalf("FATAL: "+format, args...)
}
