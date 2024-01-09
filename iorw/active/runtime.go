package active

//Runtime - interface
type Runtime interface {
	InvokeFunction(interface{}, ...interface{}) interface{}
}

type RuntimeFunc func(interface{}, ...interface{}) interface{}

func (rntmefn RuntimeFunc) InvokeFunction(event interface{}, args ...interface{}) interface{} {
	return rntmefn(event, args...)
}
