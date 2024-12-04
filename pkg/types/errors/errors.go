package errors

type ErrRetriesExceeded struct {
}

func (e *ErrRetriesExceeded) Error() string {
	return "retries exceeded"
}
