package coreerror

type ErrorLevel string

const (
	LevelInfo  ErrorLevel = "INFO"
	LevelWarn  ErrorLevel = "WARN"
	LevelError ErrorLevel = "ERROR"
)
