package download

type httpError struct {
	message  string
	canRetry bool
}

func (err *httpError) Error() string {
	return err.message
}
