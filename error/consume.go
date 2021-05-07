package error

type RpcError struct {
	Err error
}

func (e *RpcError) Error() string {
	return e.Err.Error()
}
