package utils

import (
	"fmt"
	"strings"
)

type Retryable interface {
	Retry() bool
}

type DefaultRetryable struct {
	retry bool
}

func (dr DefaultRetryable) Retry() bool {
	return dr.retry
}

func NewRetryable(retry bool) Retryable {
	return DefaultRetryable{
		retry: retry,
	}
}

type RetryableError interface {
	Retryable
	error
}

type DefaultRetryableError struct {
	Retryable
	error
}

func NewRetryableError(Retryable Retryable, err error) RetryableError {
	return &DefaultRetryableError{
		Retryable,
		err,
	}
}

type AttributeError struct {
	err string
}

func (e AttributeError) Error() string {
	return e.err
}

func NewAttributeError(err string) AttributeError {
	return AttributeError{err}
}

// Implements error
type MultiErr struct {
	errors []error
}

func (me MultiErr) Error() string {
	ret := make([]string, len(me.errors)+1)
	ret[0] = "Multiple error:"
	for ndx, err := range me.errors {
		ret[ndx+1] = fmt.Sprintf("\t%d: %s", ndx, err.Error())
	}
	return strings.Join(ret, "\n")
}

func NewMultiError(errs ...error) error {
	errors := make([]error, 0, len(errs))
	for _, err := range errs {
		if err != nil {
			errors = append(errors, err)
		}
	}
	return MultiErr{errors}
}
