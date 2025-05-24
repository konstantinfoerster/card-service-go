package aerrors

import "github.com/pkg/errors"

type ErrorType struct {
	t string
}

var (
	ErrUnknown       = ErrorType{"unknown"}       //nolint:gochecknoglobals
	ErrNotFound      = ErrorType{"not-found"}     //nolint:gochecknoglobals
	ErrInvalidInput  = ErrorType{"invalid-input"} //nolint:gochecknoglobals
	ErrAuthorization = ErrorType{"authorization"} //nolint:gochecknoglobals
)

type AppError struct {
	Key       string
	Msg       string
	Cause     error
	ErrorType ErrorType
}

func NewAuthorizationError(err error, key string) AppError {
	return AppError{
		Cause:     errors.WithStack(err),
		Key:       key,
		ErrorType: ErrAuthorization,
	}
}

func NewInvalidInputError(err error, key string, msg string) AppError {
	return AppError{
		Cause:     errors.WithStack(err),
		Key:       key,
		Msg:       msg,
		ErrorType: ErrInvalidInput,
	}
}

func NewInvalidInputMsg(key string, msg string) AppError {
	return NewInvalidInputError(errors.New(msg), key, msg)
}

func NewNotFoundError(err error, key string) AppError {
	return AppError{
		Cause:     errors.WithStack(err),
		Key:       key,
		ErrorType: ErrNotFound,
	}
}

func NewUnknownError(err error, key string) AppError {
	return AppError{
		Cause:     errors.WithStack(err),
		Key:       key,
		ErrorType: ErrUnknown,
	}
}

func (e AppError) Error() string {
	if e.Cause == nil {
		return e.Msg
	}

	return e.Cause.Error()
}

func (e AppError) Unwrap() error {
	return e.Cause
}
