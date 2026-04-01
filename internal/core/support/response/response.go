package response

type APIResponse struct {
	Status  string `json:"status"`
	Data    any    `json:"data,omitempty"`
	Error   *Error `json:"error,omitempty"`
}

type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func Success() APIResponse {
	return APIResponse{Status: "success"}
}

func SuccessWithData(data any) APIResponse {
	return APIResponse{Status: "success", Data: data}
}

func ErrorFromCode(code, message string) APIResponse {
	return APIResponse{
		Status: "error",
		Error:  &Error{Code: code, Message: message},
	}
}
