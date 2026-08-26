package domain

import "fmt"

type ErrorKind string

const (
	KindValidation ErrorKind = "validation"
	KindNotFound   ErrorKind = "not_found"
	KindConflict   ErrorKind = "conflict"
	KindState      ErrorKind = "invalid_state"
	KindFrozen     ErrorKind = "frozen"
)

type Error struct {
	Kind    ErrorKind
	Code    string
	Message string
}

func (e *Error) Error() string { return e.Message }

func NewError(kind ErrorKind, code, format string, args ...any) error {
	return &Error{Kind: kind, Code: code, Message: fmt.Sprintf(format, args...)}
}

func Validation(code, format string, args ...any) error {
	return NewError(KindValidation, code, format, args...)
}

func InvalidState(format string, args ...any) error {
	return NewError(KindState, "invalid_state", format, args...)
}

func Conflict(code, format string, args ...any) error {
	return NewError(KindConflict, code, format, args...)
}

func NotFound(resource, id string) error {
	return NewError(KindNotFound, "not_found", "%s %s 不存在", resource, id)
}

func Frozen() error {
	return NewError(KindFrozen, "campaign_frozen", "监测周期已冻结，业务数据不可修改")
}
