// Copyright 2017 Typeform SL. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Package http is a library designed to make working with the HTTP transport
// easier for jenny servers.
package http

import (
	"context"
	"crypto/rand"
	"errors"
	"net/http"
	"time"

	"github.com/jennyservices/jenny/mime"
	"github.com/oklog/ulid"
)

type contextKey int

const (
	// ContextKeyRequestMethod is populated in the context by
	// PopulateRequestContext. Its value is r.Method.
	ContextKeyRequestMethod contextKey = iota

	// ContextKeyRequestURI is populated in the context by
	// PopulateRequestContext. Its value is r.RequestURI.
	ContextKeyRequestURI

	// ContextKeyRequestURL is populated in the context by
	// PopulateRequestContext. Its value is r.RequestURI.
	ContextKeyRequestURL

	// ContextKeyRequestPath is populated in the context by
	// PopulateRequestContext. Its value is r.URL.Path.
	ContextKeyRequestPath

	// ContextKeyRequestProto is populated in the context by
	// PopulateRequestContext. Its value is r.Proto.
	ContextKeyRequestProto

	// ContextKeyRequestHost is populated in the context by
	// PopulateRequestContext. Its value is r.Host.
	ContextKeyRequestHost

	// ContextKeyRequestRemoteAddr is populated in the context by
	// PopulateRequestContext. Its value is r.RemoteAddr.
	ContextKeyRequestRemoteAddr

	// ContextKeyRequestXForwardedFor is populated in the context by
	// PopulateRequestContext. Its value is r.Header.Get("X-Forwarded-For").
	ContextKeyRequestXForwardedFor

	// ContextKeyRequestXForwardedProto is populated in the context by
	// PopulateRequestContext. Its value is r.Header.Get("X-Forwarded-Proto").
	ContextKeyRequestXForwardedProto

	// ContextKeyRequestAuthorization is populated in the context by
	// PopulateRequestContext. Its value is r.Header.Get("Authorization").
	ContextKeyRequestAuthorization

	// ContextKeyRequestReferer is populated in the context by
	// PopulateRequestContext. Its value is r.Header.Get("Referer").
	ContextKeyRequestReferer

	// ContextKeyRequestUserAgent is populated in the context by
	// PopulateRequestContext. Its value is r.Header.Get("User-Agent").
	ContextKeyRequestUserAgent

	// ContextKeyRequestXRequestID is populated in the context by
	// PopulateRequestContext. Its value is r.Header.Get("X-Request-Id").
	ContextKeyRequestXRequestID

	// ContextKeyRequestAccept is populated in the context by
	// PopulateRequestContext. Its value is r.Header.Get("Accept").
	ContextKeyRequestAccept

	// ContextKeyResponseHeaders is populated in the context whenever a
	// ServerFinalizerFunc is specified. Its value is of type http.Header, and
	// is captured only once the entire response has been written.
	ContextKeyResponseHeaders

	// ContextKeyResponseSize is populated in the context whenever a
	// ServerFinalizerFunc is specified. Its value is of type int64.
	ContextKeyResponseSize

	// ContextKeyCookies is populated with the Request cookies
	ContextKeyCookies

	// ContextKeyID is the X-Debug-ID header
	ContextKeyID

	// ContextKeyAccepts is the Accept header
	ContextKeyAccepts

	// ConetxtKeyContentType is the X-Debug-ID header
	ConetxtKeyContentType

	// ContextKeyRequestHeaders is the request headers
	ContextKeyRequestHeaders

	// ContextKeyUserAgent is the UserAgent in request
	ContextKeyUserAgent
)

// ContextCookie return a cookie that was in the http.Request
func ContextCookie(ctx context.Context, name string) *http.Cookie {
	if val := ctx.Value(ContextKeyCookies); val != nil {
		if cookies, ok := val.([]*http.Cookie); ok {
			for _, cookie := range cookies {
				if cookie.Name == name {
					return cookie
				}
			}
		}
	}
	return nil
}

func getID(r *http.Request) []byte {
	reqID := r.Header.Get("X-Request-Id")
	if reqID != "" {
		return []byte(reqID)
	}
	id := ulid.MustNew(ulid.Timestamp(time.Now()), rand.Reader)
	return id.Entropy()
}

// ContextRequestID returns a unique id for the request
func ContextRequestID(ctx context.Context) []byte {
	return ctx.Value(ContextKeyID).([]byte)
}

// ErrCouldntFindAccepts is returned when Accept mimeTypes can't be found in the header
var ErrCouldntFindAccepts = errors.New("can't find accepts in header")

// ContextAccepts returns the accept specs if the request has been originated from a HTTP request
func ContextAccepts(ctx context.Context) (mime.Types, error) {
	if accepts, ok := ctx.Value(ContextKeyAccepts).(mime.Types); ok {
		return accepts, nil
	}
	return nil, ErrCouldntFindAccepts
}

// PopulateRequestContext populates values that should travel with the request context
func PopulateRequestContext(ctx context.Context, r *http.Request) context.Context {
	for k, v := range map[contextKey]interface{}{
		ContextKeyRequestMethod:          r.Method,
		ContextKeyRequestURI:             r.RequestURI,
		ContextKeyRequestPath:            r.URL.Path,
		ContextKeyRequestURL:             r.URL,
		ContextKeyRequestProto:           r.Proto,
		ContextKeyRequestHost:            r.Host,
		ContextKeyRequestRemoteAddr:      r.RemoteAddr,
		ContextKeyRequestXForwardedFor:   r.Header.Get("X-Forwarded-For"),
		ContextKeyRequestXForwardedProto: r.Header.Get("X-Forwarded-Proto"),
		ContextKeyRequestAuthorization:   r.Header.Get("Authorization"),
		ContextKeyRequestReferer:         r.Referer(),
		ContextKeyRequestUserAgent:       r.UserAgent(),
		ContextKeyRequestXRequestID:      r.Header.Get("X-Request-Id"),
		ContextKeyRequestAccept:          r.Header.Get("Accept"),
		ContextKeyRequestHeaders:         r.Header,
		ContextKeyUserAgent:              r.UserAgent(),
		ContextKeyCookies:                r.Cookies(),
		ContextKeyID:                     getID(r),
		ContextKeyAccepts:                mime.RequestTypes(r),
	} {
		ctx = context.WithValue(ctx, k, v)
	}
	return ctx
}
