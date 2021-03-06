package query

import "github.com/suite911/error911/impl"

type Error struct {
	impl.Embed
}

func NewError(title string, cause error, msg ...interface{}) *Error {
	err := new(Error)
	err.Init(title, cause, msg...)
	return err
}

func (err *Error) Init(title string, cause error, msg ...interface{}) *Error {
	if err == nil {
		return NewError(title, cause, msg...)
	}
	err.Embed.Init(title, cause, msg...)
	return err
}

func (err *Error) Push(title string, immediateCause error, msg ...interface{}) *Error {
	if err == nil {
		return NewError(title, immediateCause, msg...)
	}
	err.Embed.Push(title, immediateCause, msg...)
	return err
}
