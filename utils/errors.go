package utils

import (
	"fmt"
	"strings"
)

//Retryable ...
type Retryable interface {
	Retry() bool
}

//DefaultRetryable ...
type DefaultRetryable struct {
	retry bool
}

//Retry ...
func (dr DefaultRetryable) Retry() bool {
	return dr.retry
}

//NewRetryable ...
func NewRetryable(retry bool) Retryable {
	return DefaultRetryable{
		retry: retry,
	}
}

//RetryableError ...
type RetryableError interface {
	Retryable
	error
}

//DefaultRetryableError ...
type DefaultRetryableError struct {
	Retryable
	error
}

//NewRetryableError ...
func NewRetryableError(Retryable Retryable, err error) RetryableError {
	return &DefaultRetryableError{
		Retryable,
		err,
	}
}

//AttributeError ...
type AttributeError struct {
	err string
}

//Error ...
func (e AttributeError) Error() string {
	return e.err
}

//NewAttributeError ...
func NewAttributeError(err string) AttributeError {
	return AttributeError{err}
}

// MultiErr implements error
type MultiErr struct {
	errors []error
}

//Error ...
func (me MultiErr) Error() string {
	ret := make([]string, len(me.errors)+1)
	ret[0] = "Multiple error:"
	for ndx, err := range me.errors {
		ret[ndx+1] = fmt.Sprintf("\t%d: %s", ndx, err.Error())
	}
	return strings.Join(ret, "\n")
}

//NewMultiError ...
func NewMultiError(errs ...error) error {
	errors := make([]error, 0, len(errs))
	for _, err := range errs {
		if err != nil {
			errors = append(errors, err)
		}
	}
	return MultiErr{errors}
}
