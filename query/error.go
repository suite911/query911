package query

import "github.com/suite911/error911"

type Error struct {
	error911.Error
}

func NewError() *Error {
	return new(Error).Init()
}

func (e *Error) Init() *Error {
	return e
}
