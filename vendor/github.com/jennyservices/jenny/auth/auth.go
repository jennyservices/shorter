// Copyright 2017 Typeform SL. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Package auth defines how authorization works between middlewares by default.
package auth

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/pkg/errors"

	stdjwt "github.com/dgrijalva/jwt-go"
	kitjwt "github.com/go-kit/kit/auth/jwt"
	"github.com/go-kit/kit/endpoint"
	jennyerrors "github.com/jennyservices/jenny/errors"
	jennyhttp "github.com/jennyservices/jenny/http"
)

type contextKey int

const (
	// JWTContextKey is the context key for JWT
	JWTContextKey contextKey = iota

	// UserContextKey is the context key for a user, this requires
	// a middleware with the userfunc to be present
	UserContextKey

	// BasicContextKey is the context key for basic auth, it returns a username and password
	BasicContextKey

	// ScopesContextKey is the context key for scopes present in a context
	ScopesContextKey
)

var (
	// ErrUserNotFoundInContext is returned when a user is not found in the
	//context under the AuthUser key
	ErrUserNotFoundInContext = jennyerrors.NewHTTPError(errors.New("user not found in context"), http.StatusUnauthorized)

	// ErrJWTNotFoundInContext is the error returned when a JWT is not present in
	// the context under the JWTContextKey key
	ErrJWTNotFoundInContext = jennyerrors.NewHTTPError(errors.New("jwt not found in context"), http.StatusUnauthorized)

	// ErrScopesNotFoundInContext is the error returned when scopes are not present
	// in the context under the ScopesContextKey
	ErrScopesNotFoundInContext = jennyerrors.NewHTTPError(errors.New("scopes not found in context"), http.StatusForbidden)

	// ErrAuthNotAuthorized is the error returned when the request doesn't have enough permissions
	ErrAuthNotAuthorized = jennyerrors.NewHTTPError(errors.New("request does not have sufficent permissions to continue"), http.StatusForbidden)
)

// JWTScopesExtrator takes jwt.MapClaims and extracts the requests scopes from it
type JWTScopesExtrator func(stdjwt.MapClaims) ([]string, error)

// ScopesToContext takes claims and extracts scopes from it to inject it
// to the context.
// this middleware assumes that the gokit jwt.Middlewares are used and the
// JWTClaimscontextKey is present
func ScopesToContext(claimsScopes JWTScopesExtrator) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			if token, ok := ctx.Value(kitjwt.JWTClaimsContextKey).(*stdjwt.Token); ok {
				if mapClaims, ok := token.Claims.(stdjwt.MapClaims); ok {
					user, err := claimsScopes(mapClaims)
					if err == nil {
						ctx = context.WithValue(ctx, ScopesContextKey, user)
					}
				}
			}
			return next(ctx, request)
		}
	}
}

// User represents the minimum a user object should implement
// The UniqueID function should return a slice of bytes that are unique.
//
// In the case that the underlying object has a numerical ID the implementor should
// convert the numerical value to a byte slice like so;
//
//	func (u *User) UniqueID() []byte {
// 		buf := make([]byte, binary.MaxVarintLen64)
// 		n := binary.PutUvarint(buf, u.ID)
// 		return buf
//	}
//
// In the case of the ID being a string the implementor should make  sure the capitalization
// of the string is consistent.
// Jenny will threat 0xDEADBEEF != 0xdeadbeef as different IDs.
type User interface {
	UniqueID() []byte
}

// ExtendedUser encapsulates more information that User, while the User inferface has
// actual practical use, ExtendedUser is purely for convinience
type ExtendedUser interface {
	User
	Email() string                // Email returns an email for communicating with the User
	DisplayName() (string, error) // DisplayName is used when you need to address the user, this is here for convinience
	Details() map[string]string   // Returns details for the user that aren't documented like id and email
}

// JWTToContext takes a JWTUserExtractor function and injects the User as
func JWTToContext(keyFunc stdjwt.Keyfunc, method stdjwt.SigningMethod, newClaims kitjwt.ClaimsFactory) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			tokenString, ok := ctx.Value(kitjwt.JWTTokenContextKey).(string)
			if !ok {
				return next(ctx, request)
			}

			// Parse takes the token string and a function for looking up the
			// key. The latter is especially useful if you use multiple keys
			// for your application.  The standard is to use 'kid' in the head
			// of the token to identify which key to use, but the parsed token
			// (head and claims) is provided to the callback, providing
			// flexibility.
			token, err := stdjwt.ParseWithClaims(tokenString, newClaims(), keyFunc)
			if err != nil {
				log.Println(errors.Wrap(err, "jwttoctx"))
			}
			if !token.Valid {
				return next(ctx, request)
			}

			ctx = context.WithValue(ctx, kitjwt.JWTClaimsContextKey, token.Claims)
			return next(ctx, request)
		}
	}
}

// JWTUserExtractor extracts the user from jwt.MapClaims
type JWTUserExtractor func(stdjwt.Claims) (User, error)

// UserToContext takes a JWTUserExtractor function and injects the User as
func UserToContext(claimsUser JWTUserExtractor) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			if claims, ok := ctx.Value(kitjwt.JWTClaimsContextKey).(stdjwt.Claims); ok {
				user, err := claimsUser(claims)
				if err == nil {
					ctx = context.WithValue(ctx, UserContextKey, user)
				}

			}
			return next(ctx, request)
		}
	}
}

// RequireScopes protects an endpoint that requires scopes to be present
func RequireScopes(scopes []string) endpoint.Middleware {
	x := make(map[string]bool)
	for _, scope := range scopes {
		x[scope] = false
	}
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (response interface{}, err error) {
			scopes, ok := ctx.Value(ScopesContextKey).([]string)
			if !ok {
				scopes = []string{}
			}
			checkList := x
			for _, scope := range scopes {
				checkList[scope] = true
			}
			hasAccess := true
			missingScopes := []string{}

			for scope, checked := range checkList {
				hasAccess = hasAccess && checked
				missingScopes = append(missingScopes, scope)
			}
			if !hasAccess {
				return nil, fmt.Errorf("request is missing these scopes: %s", strings.Join(missingScopes, ", "))
			}
			return next(ctx, request)
		}
	}
}

// ContextJWT returns the jwt if one exists in context
func ContextJWT(ctx context.Context) (*stdjwt.Token, error) {
	jwtoken, ok := ctx.Value(kitjwt.JWTClaimsContextKey).(*stdjwt.Token)
	if !ok {
		return nil, errors.Wrap(ErrJWTNotFoundInContext, "jwt to context")
	}
	return jwtoken, nil
}

// ContextUser returns an object that implements the user interface
func ContextUser(ctx context.Context) (User, error) {
	u, ok := ctx.Value(UserContextKey).(User)
	if !ok {
		return nil, ErrUserNotFoundInContext
	}
	return u, nil
}

// BasicAuth returns the username and password provided in the request's
// Authorization header, if the request uses HTTP Basic Authentication.
// See RFC 2617, Section 2.
func BasicAuth(ctx context.Context) (username, password string, ok bool) {
	if auth, ok := ctx.Value(jennyhttp.ContextKeyRequestAuthorization).(string); ok && auth != "" {
		return parseBasicAuth(auth)
	}
	return
}

// parseBasicAuth parses an HTTP Basic Authentication string.
// "Basic QWxhZGRpbjpvcGVuIHNlc2FtZQ==" returns ("Aladdin", "open sesame", true).
func parseBasicAuth(auth string) (username, password string, ok bool) {
	const prefix = "Basic "
	if !strings.HasPrefix(auth, prefix) {
		return
	}
	c, err := base64.StdEncoding.DecodeString(auth[len(prefix):])
	if err != nil {
		return
	}
	cs := string(c)
	s := strings.IndexByte(cs, ':')
	if s < 0 {
		return
	}
	return cs[:s], cs[s+1:], true
}
