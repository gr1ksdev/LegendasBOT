package mtproto

import "errors"

// Erros especificos do MTProto.
var (
	ErrCodeInvalid         = errors.New("código inválido")
	ErrCodeExpired         = errors.New("código expirado")
	ErrPasswordIncorrect   = errors.New("senha incorreta")
	ErrFloodWait           = errors.New("FloodWait: muitas requisições")
	ErrSessionInvalid      = errors.New("sessão inválida")
	ErrSessionExpired      = errors.New("sessão expirada")
	ErrAccountBanned       = errors.New("conta banida")
	ErrAccountRemoved      = errors.New("conta removida")
	ErrNetworkFailure      = errors.New("falha de rede")
	ErrPhoneNumberInvalid  = errors.New("número de telefone inválido")
	ErrPhoneNumberBanned   = errors.New("número de telefone banido")
	ErrPhoneNumberOccupied = errors.New("número já está em uso em outra conta")
)

// IsRetryableError retorna true se o erro pode ser tratado com retry.
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, ErrFloodWait) || errors.Is(err, ErrNetworkFailure)
}

// IsAuthError retorna true se o erro esta relacionado a autenticacao.
func IsAuthError(err error) bool {
	if err == nil {
		return false
	}
	return errors.Is(err, ErrCodeInvalid) ||
		errors.Is(err, ErrCodeExpired) ||
		errors.Is(err, ErrPasswordIncorrect) ||
		errors.Is(err, ErrSessionInvalid) ||
		errors.Is(err, ErrSessionExpired)
}
