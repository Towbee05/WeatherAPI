package errors

type ErrorStruct struct {
	statusCode int
	message    string
	err        error
}

func FailOnError(statusCode int, message string, err error) ErrorStruct {
	return ErrorStruct{
		statusCode: statusCode,
		message:    message,
		err:        err,
	}
}
