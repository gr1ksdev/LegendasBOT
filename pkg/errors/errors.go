package errors

import (
	"fmt"
	"net/http"
)

type AppError struct {
	Code    int    `json:"-"`
	Message string `json:"message"`
	Type    string `json:"type,omitempty"`
}

func (e *AppError) Error() string {
	return e.Message
}

func New(code int, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

// Erros comuns pré-definidos
var (
	ErrNotFound     = New(http.StatusNotFound, "Recurso não encontrado")
	ErrUnauthorized = New(http.StatusUnauthorized, "Não autorizado")
	ErrForbidden    = New(http.StatusForbidden, "Acesso negado")
	ErrBadRequest   = New(http.StatusBadRequest, "Dados inválidos")
	ErrConflict     = New(http.StatusConflict, "Conflito")
	ErrInternal     = New(http.StatusInternalServerError, "Erro interno do servidor")
)

func BadRequest(msg string) *AppError {
	return New(http.StatusBadRequest, msg)
}

func Internal(err error) *AppError {
	return New(http.StatusInternalServerError, fmt.Sprintf("Erro interno: %v", err))
}
