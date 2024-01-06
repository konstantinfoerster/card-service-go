package common

import "github.com/pkg/errors"

type ErrorType struct {
	t string
}

var (
	ErrTypeUnknown       = ErrorType{"unknown"}       //nolint:gochecknoglobals
	ErrTypeInvalidInput  = ErrorType{"invalid-input"} //nolint:gochecknoglobals
	ErrTypeAuthorization = ErrorType{"authorization"} //nolint:gochecknoglobals
)

type AppError struct {
	Key       string
	Msg       string
	Cause     error
	ErrorType ErrorType
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

func NewAuthorizationError(err error, key string) AppError {
	return AppError{
		Cause:     errors.WithStack(err),
		Key:       key,
		ErrorType: ErrTypeAuthorization,
	}
}

func NewInvalidInputError(err error, key string, msg string) AppError {
	return AppError{
		Cause:     errors.WithStack(err),
		Key:       key,
		Msg:       msg,
		ErrorType: ErrTypeInvalidInput,
	}
}

func NewInvalidInputMsg(key string, msg string) AppError {
	return AppError{
		Cause:     errors.New(msg),
		Key:       key,
		Msg:       msg,
		ErrorType: ErrTypeInvalidInput,
	}
}

func NewUnknownError(err error, key string) AppError {
	return AppError{
		Cause:     errors.WithStack(err),
		Key:       key,
		ErrorType: ErrTypeUnknown,
	}
}
