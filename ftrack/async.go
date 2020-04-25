package ftrack

type AsyncQueryResult struct {
	Result *QueryResult
	Err    error
}

func (session *Session) AsyncQuery(expression string) <-chan AsyncQueryResult {
	ch := make(chan AsyncQueryResult)
	push := func(r *QueryResult, err error) { ch <- AsyncQueryResult{r, err} }
	go func() { defer close(ch); push(session.Query(expression)) }()
	return ch
}

type AsyncCallResult struct {
	Result []interface{}
	Err    error
}

func (session *Session) AsyncCall(operations ...interface{}) <-chan AsyncCallResult {
	ch := make(chan AsyncCallResult)
	push := func(r []interface{}, err error) { ch <- AsyncCallResult{r, err} }
	go func() { defer close(ch); push(session.Call(operations...)) }()
	return ch
}
