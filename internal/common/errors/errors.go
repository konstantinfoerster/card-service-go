package errors

type ErrorType struct {
	t string
}

var (
	ErrorTypeUnknown = ErrorType{"unknown"}
)

type Error struct {
	error     error
	key       string
	errorType ErrorType
}

func (e Error) Error() string {
	return e.error.Error()
}

func (e Error) Key() string {
	return e.key
}

func (e Error) Type() ErrorType {
	return e.errorType
}

func NewError(error error, key string) Error {
	return Error{
		error:     error,
		key:       key,
		errorType: ErrorTypeUnknown,
	}
}
