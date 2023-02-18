package common

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
	Err       error
	ErrorType ErrorType
}

func (e AppError) Error() string {
	if e.Err == nil {
		return e.Msg
	}

	return e.Err.Error()
}

func (e AppError) Unwrap() error {
	return e.Err
}

func NewAuthorizationError(err error, key string) AppError {
	return AppError{
		Err:       err,
		Key:       key,
		ErrorType: ErrTypeAuthorization,
	}
}

func NewInvalidInputError(err error, key string, msg string) AppError {
	return AppError{
		Err:       err,
		Key:       key,
		Msg:       msg,
		ErrorType: ErrTypeInvalidInput,
	}
}

func NewUnknownError(err error, key string) AppError {
	return AppError{
		Err:       err,
		Key:       key,
		ErrorType: ErrTypeUnknown,
	}
}
