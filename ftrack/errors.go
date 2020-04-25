package ftrack

import "fmt"

type DecodeError struct {
	msg  string
	data interface{}
}

func (error *DecodeError) Error() string {
	return fmt.Sprintf("decode error: %s on data: %s", error.msg, error.data)
}

type EncodeError struct {
	msg  string
	data interface{}
}

func (error *EncodeError) Error() string {
	return fmt.Sprintf("encode error %s on data: %s", error.msg, error.data)
}

type ServerError struct {
	Msg       string
	ErrorCode int
	Exception string
}

func (error *ServerError) Error() string {
	return fmt.Sprintf("ServerError: %s - %s code: %d", error.Exception, error.Msg, error.ErrorCode)
}

type ServerValidationError ServerError

func (error *ServerValidationError) Error() string {
	return fmt.Sprintf("ServerValidationError: %s - %s code: %d", error.Exception, error.Msg, error.ErrorCode)
}

type ServerPermissionDeniedError ServerError

func (error *ServerPermissionDeniedError) Error() string {
	return fmt.Sprintf("ServerPermissionDeniedError: %s - %s code: %d", error.Exception, error.Msg, error.ErrorCode)
}

type MalformedResponseError struct {
	Content []byte
}

func (error *MalformedResponseError) Error() string {
	return fmt.Sprintf("MalformedResponseError: content: %s", error.Content)
}
