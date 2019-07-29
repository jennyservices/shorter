// Copyright 2017 Typeform SL. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Package errors defines some sensible defaults for jenny generated services
package errors

import (
	"context"
	"errors"
	"log"
	"net/http"

	"github.com/jennyservices/jenny/encoders"
	"github.com/jennyservices/jenny/mime"
	"github.com/go-kit/kit/endpoint"
	kitthttp "github.com/go-kit/kit/transport/http"
)

// New is the std lib errrors.New
var New = errors.New

// ErrorReporter is a special middleware that works similary to the tracing middleware,
// it requires what the operationID should be inorder to report it's errors
func ErrorReporter(reporter Reporter, op string) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {
			var err error
			var i interface{}
			defer reporter.Report(ctx, err, op)
			i, err = next(ctx, request)
			return i, err
		}
	}
}

// DefaultErrorEncoder is the default jenny encoder for errors. You can change this
// setting in the options package
func DefaultErrorEncoder(ctx context.Context, err error, w http.ResponseWriter) {
	log.Println(err)
	newEncoder, mt, encErr := encoders.ResponseEncoder(ctx,
		[]mime.Type{
			mime.ApplicationJSON,
			mime.TextPlain,
		})
	var enc encoders.Encoder
	if encErr != nil {
		enc = encoders.TextEncoder(w)
	} else {
		w.Header().Set("Content-Type", string(mt))
		enc = newEncoder(w)
	}

	if httperr, ok := err.(HTTPError); ok {
		w.WriteHeader(httperr.StatusCode())
	}
	enc.Encode(err)
	return
}

// Reporter is an interface used to report errors to an error reporting service
// like sentry or rollbar
type Reporter interface {
	Report(context.Context, error, string)
}

// NoopReporter is the default reporter, it does nothing
type NoopReporter struct{}

// Report does nothing
func (NoopReporter) Report(context.Context, error, string) {}

// HTTPError error is an interface to signal jenny whether a error should be
// displayed as is to the public or be obfuscated
type HTTPError interface {
	error
	kitthttp.StatusCoder
}

type httpError struct {
	code int
	err  error
}

// NewHTTPError wraps an err with a HTTP status code
func NewHTTPError(err error, code int) HTTPError {
	return &httpError{
		err:  err,
		code: code,
	}
}

func (he *httpError) Error() string {
	return he.err.Error()
}

func (he *httpError) StatusCode() int {
	return he.code
}
