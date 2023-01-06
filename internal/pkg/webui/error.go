package webui

import (
	"fmt"
	"net/http"
)

type StatusError interface {
	error
	Status() int
}

type statusErr struct {
	error
	status int
}

func (e statusErr) Status() int { return e.status }

func NewError(status int, format string, args ...any) StatusError {
	return statusErr{error: fmt.Errorf(format, args...), status: status}
}

func HTTPError(w http.ResponseWriter, err error) {
	switch err := err.(type) {
	case StatusError:
		http.Error(w, err.Error(), err.Status())
	case error:
		http.Error(w, err.Error(), http.StatusInternalServerError)
	case nil:
		// No-op
	}
}
