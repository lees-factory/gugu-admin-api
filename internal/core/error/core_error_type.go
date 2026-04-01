package coreerror

type ErrorType struct {
	Kind    ErrorKind
	Code    string
	Message string
	Level   ErrorLevel
}

// Admin (A1xxx)
var (
	InvalidRequest   = ErrorType{KindClient, A1000, "잘못된 요청입니다", LevelInfo}
	ResourceNotFound = ErrorType{KindClient, A1001, "리소스를 찾을 수 없습니다", LevelInfo}
)

// Auth (A2xxx)
var (
	UnauthorizedAccess = ErrorType{KindUnauthorized, A2000, "인증이 필요합니다", LevelWarn}
)

// Internal (A9xxx)
var (
	InternalError = ErrorType{KindServer, A9000, "서버 내부 오류가 발생했습니다", LevelError}
)
