package domain

import "fmt"

type ErrorKind string

const (
	ErrorValidation  ErrorKind = "validation_error"
	ErrorConflict    ErrorKind = "state_conflict"
	ErrorVersion     ErrorKind = "version_conflict"
	ErrorNotFound    ErrorKind = "not_found"
	ErrorIdempotency ErrorKind = "idempotency_conflict"
	ErrorQuery       ErrorKind = "invalid_query"
	ErrorIntegrity   ErrorKind = "data_integrity_error"
)

type Error struct {
	Kind    ErrorKind
	Field   string
	Message string
}

func (e *Error) Error() string {
	if e.Field == "" {
		return e.Message
	}
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

func Validation(field, message string) error {
	return &Error{Kind: ErrorValidation, Field: field, Message: message}
}
func Conflict(message string) error { return &Error{Kind: ErrorConflict, Message: message} }
func VersionConflict() error {
	return &Error{Kind: ErrorVersion, Message: "expectedVersion 与当前版本不一致"}
}
func NotFound(entity string) error { return &Error{Kind: ErrorNotFound, Message: entity + "不存在"} }
func IdempotencyConflict() error {
	return &Error{Kind: ErrorIdempotency, Message: "idempotencyKey 已用于不同请求"}
}
func QueryError(field, message string) error {
	return &Error{Kind: ErrorQuery, Field: field, Message: message}
}
func IntegrityError(message string) error {
	return &Error{Kind: ErrorIntegrity, Message: message}
}
