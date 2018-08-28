package query

type Error struct {
	error911.E911Impl
}

func NewError() *Error {
	return new(Error).Init()
}

func (e *Error) Init() *Error {
	e.E911Impl.Init("Query Error")
	return e
}