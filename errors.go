package apitoolkit

import (
	"context"
	"log"
	"time"

	"github.com/go-errors/errors"
)

type ctxKey string

var ErrorListCtxKey = ctxKey("error-list")

// ATError is the Apitoolkit error type/object
type ATError struct {
	When 	time.Time
	ErrorType string 
	Message string 
	StackTrace string
}

// ReportError Allows you to report an error from your server to APIToolkit.
// This error would be associated with a given request, 
// and helps give a request more context especially when investigating incidents
func ReportError(ctx context.Context, err error) {
	if err == nil{
		return
	}

	errorList, ok := ctx.Value(ErrorListCtxKey).(*[]ATError)
	if !ok {
		log.Printf("APIToolkit: ErrorList context key was not found in the context. Is the middleware configured correctly? Error will not be notified. Error: %v \n", err)
		return
	}

	errW := errors.Wrap(err, 1)
	*errorList = append(*errorList, ATError{
		When: time.Now(),
		ErrorType: errW.TypeName(),
		Message: errW.Error(),
		StackTrace: errW.ErrorStack(),	
	})
}
