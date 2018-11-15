package cpio

type Logger int
func (l *Logger) Debug(format string, args ...interface{}) {}
func (l *Logger) Debugf(format string, args ...interface{}) {}

var logger Logger
