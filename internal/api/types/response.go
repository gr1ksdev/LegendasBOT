package types

type APIResponse[T any] struct {
	Success bool   `json:"success"`
	Message string `json:"message,omitempty"`
	Data    T      `json:"data,omitempty"`
}

func NewSuccessResponse[T any](data T, message ...string) APIResponse[T] {
	msg := ""
	if len(message) > 0 {
		msg = message[0]
	}
	return APIResponse[T]{
		Success: true,
		Message: msg,
		Data:    data,
	}
}

func NewErrorResponse(message string) APIResponse[any] {
	return APIResponse[any]{
		Success: false,
		Message: message,
	}
}
