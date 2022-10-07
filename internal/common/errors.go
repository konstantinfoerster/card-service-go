package common

type ErrorType struct {
	t string
}

var (
	ErrorTypeUnknown = ErrorType{"unknown"}
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

func NewUnknownError(error error, key string) AppError {
	return AppError{
		Err:       error,
		Key:       key,
		ErrorType: ErrorTypeUnknown,
	}
}
