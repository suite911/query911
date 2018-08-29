package query

import "github.com/suite911/error911"

type Error *error911.SError;

func New(title string, cause error, msg ...interface{}) Error {
	return error911.ImplNew(title, cause, msg...)
}

func (err Error) Init(title string, cause error, msg ...interface{}) Error {
	return error911.ImplInit(err, title, cause, msg...)
}

func (err Error) New(title string, cause error, msg ...interface{}) Error {
	return error911.ImplNewMethod(err, title, cause, msg...)
}
func (err *Error) Push(title string, cause error, msg ...interface{}) {
	error911.ImplPush(err, title, cause, msg...)
}
