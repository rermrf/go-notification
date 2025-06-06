package logger

type NopLogger struct{}

func (n NopLogger) With(args ...Field) Logger {
	return n
}

func NewNopLogger() *NopLogger {
	return &NopLogger{}
}

func (n NopLogger) Debug(msg string, args ...Field) {
}

func (n NopLogger) Info(msg string, args ...Field) {
}

func (n NopLogger) Warn(msg string, args ...Field) {
}

func (n NopLogger) Error(msg string, args ...Field) {
}
