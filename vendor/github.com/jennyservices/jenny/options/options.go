// Copyright 2017 Typeform SL. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package options

import (
	"github.com/jennyservices/jenny/auth"
	"github.com/jennyservices/jenny/errors"
	jennyhttp "github.com/jennyservices/jenny/http"
	stdjwt "github.com/dgrijalva/jwt-go"
	"github.com/go-kit/kit/auth/jwt"
	"github.com/go-kit/kit/endpoint"
	kittracing "github.com/go-kit/kit/tracing/opentracing"
	kitthttp "github.com/go-kit/kit/transport/http"
	"github.com/golang-collections/collections/stack"
	opentracing "github.com/opentracing/opentracing-go"
)

// Options represents a set of options that can be passed to Jenny Service Interfaces with
// the implementation of the interface
type Options struct {
	tracer opentracing.Tracer

	errorReporter errors.Reporter
	errorEncoder  kitthttp.ErrorEncoder

	kf stdjwt.Keyfunc
	sm stdjwt.SigningMethod
	cf jwt.ClaimsFactory

	jwtFunc kitthttp.RequestFunc

	userFunc   auth.JWTUserExtractor
	scopesFunc auth.JWTScopesExtrator

	middlewares map[string]*stack.Stack
}

// Option represnets a option for making middlewares like tracing and reporting
// middlewares
type Option func(*Options)

// New retunrs new options with defail noop tracer and no op error reporter
func New() *Options {
	return &Options{
		tracer:        opentracing.NoopTracer{},
		errorReporter: errors.NoopReporter{},
		errorEncoder:  errors.DefaultErrorEncoder,
		middlewares:   make(map[string]*stack.Stack),
	}
}

// WithTracing gets a opentracing.Tracer and injects it to every endpoint as
// a tracing middleware
func WithTracing(tracer opentracing.Tracer) Option {
	return func(m *Options) { m.tracer = tracer }
}

// OpMiddlewares returns the middlewares that were in options
// Current ordering of the middlewares goes as follows
//
// 	Request
// 		↓
// 	Requests-ID (enabled by default)
// 		↓
// 	Tracing (enabled noop by default)
// 		↓
// 	Error reporting (enabled noop by default)
// 		↓
// 	JWT parser (disabled by default, enable by passing WithJWTParser)
// 		↓
// 	User parser (disabled by default, enable by passing WithUserParser); (this is useful for ratelimiting by user)
// 		↓
// 	Scopes parser (disabled by default, enable by passing WithScopesParser)
func (m *Options) OpMiddlewares(operation string) endpoint.Middleware {
	var operationStack *stack.Stack
	if st, ok := m.middlewares[operation]; ok {
		operationStack = st
	} else {
		operationStack = stack.New()
	}

	if m.scopesFunc != nil {
		operationStack.Push(auth.ScopesToContext(m.scopesFunc))
	}
	if m.userFunc != nil {
		operationStack.Push(auth.UserToContext(m.userFunc))
	}
	if m.kf != nil && m.sm != nil && m.cf != nil {
		operationStack.Push(auth.JWTToContext(m.kf, m.sm, m.cf))
	}
	operationStack.Push(kittracing.TraceClient(m.tracer, operation))
	operationStack.Push(errors.ErrorReporter(m.errorReporter, operation))

	mware := operationStack.Pop().(endpoint.Middleware)

	for i := operationStack.Len(); i > 0; i-- {
		mware = endpoint.Chain(mware, operationStack.Pop().(endpoint.Middleware))
	}
	return mware
}

// RegisterMiddleware pushed the middleware to the middlewareStack, the order in
// whihc you register the middlewares determines it's execution order, except
// for middlewares you register withFoo methods.
func (m *Options) RegisterMiddleware(op string, middlewares ...endpoint.Middleware) {
	if _, ok := m.middlewares[op]; !ok {
		m.middlewares[op] = stack.New()
	}
	for _, mw := range middlewares {
		m.middlewares[op].Push(mw)
	}
}

// HTTPOptions returns all the server options to be used with HTTP endpoints
func (m *Options) HTTPOptions() []kitthttp.ServerOption {
	opts := []kitthttp.ServerOption{
		kitthttp.ServerBefore(jennyhttp.PopulateRequestContext),
	}
	if m.jwtFunc != nil {
		opts = append(opts, kitthttp.ServerBefore(m.jwtFunc))
	}
	if m.errorEncoder != nil {
		opts = append(opts, kitthttp.ServerErrorEncoder(m.errorEncoder))
	}
	return opts
}

// WithErrorReporting takes a error reporter like sentry or rollbar and adds it
// as a middleware so the errors that aren't returned to client
func WithErrorReporting(reporter errors.Reporter) Option {
	return func(m *Options) {
		m.errorReporter = reporter
	}
}

// WithJWTParser gets a keyfunc method and claims factory and injects it to the enpoints that require
// JWT security
func WithJWTParser(jwtFunc kitthttp.RequestFunc, keyFunc stdjwt.Keyfunc, method stdjwt.SigningMethod, cf jwt.ClaimsFactory) Option {
	return func(m *Options) {
		m.jwtFunc = jwtFunc
		m.kf = keyFunc
		m.sm = method
		m.cf = cf
	}
}

// WithScopesParser adds a middleware that injects Scopes in the context
// see https://godoc.org/github.com/jennyservices/jenny/auth for docs
func WithScopesParser(scopesFunc auth.JWTScopesExtrator) Option {
	return func(m *Options) {
		m.scopesFunc = scopesFunc
	}
}

// WithUserParser adds a middleware that injects an object that implents the
// User interface in the context see https://godoc.org/github.com/jennyservices/jenny/auth for docs
func WithUserParser(userFunc auth.JWTUserExtractor) Option {
	return func(m *Options) {
		m.userFunc = userFunc
	}
}

// WithErrorEncoder sets the errorencoding options for http endpoints
func WithErrorEncoder(errorEncoder kitthttp.ErrorEncoder) Option {
	return func(m *Options) {
		m.errorEncoder = errorEncoder
	}
}
