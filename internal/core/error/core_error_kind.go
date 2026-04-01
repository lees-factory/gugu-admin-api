package coreerror

type ErrorKind string

const (
	KindClient       ErrorKind = "CLIENT_ERROR"
	KindServer       ErrorKind = "SERVER_ERROR"
	KindUnauthorized ErrorKind = "UNAUTHORIZED"
)
