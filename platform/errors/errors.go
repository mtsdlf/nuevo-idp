package errors

import "errors"

// Kind clasifica errores para poder mapearlos después a HTTP, métricas, etc.
type Kind string

const (
	KindDomain     Kind = "domain"
	KindValidation Kind = "validation"
	KindConflict   Kind = "conflict"
	KindNotFound   Kind = "not_found"
	KindInternal   Kind = "internal"
)

// Error es un wrapper enriquecido con Kind y Code.
type Error struct {
	Kind    Kind   // qué tipo de error es (domain, validation, ...)
	Code    string // código estable, apto para logs/metrics (ej: "application_already_active")
	Message string // mensaje pensado para humanos
	Err     error  // causa original (opcional)
}

func (e *Error) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

func (e *Error) Unwrap() error { return e.Err }

// Helpers de construcción.

func Domain(code, msg string, cause error) *Error {
	return &Error{Kind: KindDomain, Code: code, Message: msg, Err: cause}
}

func Validation(code, msg string, cause error) *Error {
	return &Error{Kind: KindValidation, Code: code, Message: msg, Err: cause}
}

func Conflict(code, msg string, cause error) *Error {
	return &Error{Kind: KindConflict, Code: code, Message: msg, Err: cause}
}

func NotFound(code, msg string, cause error) *Error {
	return &Error{Kind: KindNotFound, Code: code, Message: msg, Err: cause}
}

func Internal(code, msg string, cause error) *Error {
	return &Error{Kind: KindInternal, Code: code, Message: msg, Err: cause}
}

// IsKind permite preguntar si, al desempaquetar err, aparece un Error de cierto Kind.
func IsKind(err error, kind Kind) bool {
	var e *Error
	if errors.As(err, &e) {
		return e.Kind == kind
	}
	return false
}

// Code devuelve el código estable si err es un Error de plataforma.
func Code(err error) string {
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return ""
}
