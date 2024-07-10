package apitoolkit

import (
	"context"
	"errors"
	"log"
	"reflect"
	"time"

	gerrors "github.com/go-errors/errors"
)

type ctxKey string

var (
	ErrorListCtxKey         = ctxKey("error-list")
	CurrentRequestMessageID = ctxKey("current-req-msg-id")
	CurrentClient           = ctxKey("current=apitoolkit-client")
)

// ATError is the Apitoolkit error type/object
type ATError struct {
	When             time.Time `json:"when,omitempty"`
	ErrorType        string    `json:"error_type,omitempty"`
	RootErrorType    string    `json:"root_error_type,omitempty"`
	Message          string    `json:"message,omitempty"`
	RootErrorMessage string    `json:"root_error_message,omitempty"`
	StackTrace       string    `json:"stack_trace,omitempty"`
}

// ReportError Allows you to report an error from your server to APIToolkit.
// This error would be associated with a given request,
// and helps give a request more context especially when investigating incidents
func ReportError(ctx context.Context, err error) {
	if err == nil {
		return
	}

	errorList, ok := ctx.Value(ErrorListCtxKey).(*[]ATError)
	if !ok {
		log.Printf("APIToolkit: ErrorList context key was not found in the context. Is the middleware configured correctly? Error will not be notified. Error: %v \n", err)
		return
	}

	*errorList = append(*errorList, BuildError(err))
}

func BuildError(err error) ATError {
	errType := reflect.TypeOf(err).String()

	rootError := rootCause(err)
	rootErrorType := reflect.TypeOf(rootError).String()
	errW := gerrors.Wrap(err, 2)
	return ATError{
		When:             time.Now(),
		ErrorType:        errType,
		RootErrorType:    rootErrorType,
		RootErrorMessage: rootError.Error(),
		Message:          errW.Error(),
		StackTrace:       errW.ErrorStack(),
	}
}

// rootCause recursively unwraps an error and returns the original cause.
func rootCause(err error) error {
	for {
		cause := errors.Unwrap(err)
		if cause == nil {
			return err
		}
		err = cause
	}
}
