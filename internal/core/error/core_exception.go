package coreerror

type CoreException struct {
	Type    ErrorType
	Data    any
	Message string
}

func New(errorType ErrorType) *CoreException {
	return &CoreException{
		Type:    errorType,
		Message: errorType.Message,
	}
}

func NewWithData(errorType ErrorType, data any) *CoreException {
	return &CoreException{
		Type:    errorType,
		Data:    data,
		Message: errorType.Message,
	}
}

func (e *CoreException) Error() string {
	return e.Message
}

func (e *CoreException) Is(target error) bool {
	t, ok := target.(*CoreException)
	if !ok {
		return false
	}
	return e.Type.Code == t.Type.Code
}
