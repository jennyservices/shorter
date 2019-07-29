// Copyright 2017 Typeform SL. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Package encoders is a set of encoders to be used with Jenny
package encoders

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"

	jennyhttp "github.com/jennyservices/jenny/http"
	"github.com/jennyservices/jenny/mime"
	"github.com/go-kit/kit/endpoint"
	"github.com/golang/gddo/httputil/header"
	"github.com/gorilla/schema"
	"github.com/pkg/errors"
)

// Encoder returns encoder that wraps around
type Encoder interface {
	Encode(v interface{}) error
}

type NewEncoderFunc func(io.Writer) Encoder

type plainTextEncoder struct {
	w io.Writer
}

func (p *plainTextEncoder) Encode(v interface{}) error {
	p.w.Write([]byte(fmt.Sprint(v)))
	return nil
}

type redirectEncoder struct {
	w http.ResponseWriter
}

func (u *redirectEncoder) Encode(v interface{}) error {
	return nil
}

type formEncoder struct {
	w io.Writer
}

func (f *formEncoder) Encode(v interface{}) error {
	values := make(url.Values)
	enc := schema.NewEncoder()
	if err := enc.Encode(v, values); err != nil {
		return err
	}
	f.w.Write([]byte(values.Encode()))
	return nil
}

type byteEncoder struct {
	w io.Writer
}

var ErrByteEncoderNotSupported = errors.New("unsupported interface for byte encoder")

func (b *byteEncoder) Encode(v interface{}) error {
	switch v.(type) {
	case []byte:
		bytez := v.([]byte)
		b.w.Write(bytez)
		return nil
	default:
		return ErrByteEncoderNotSupported
	}
}

var (
	ErrEncoderNotFound = errors.New("unsupported media response in Accept")
	JSONEncoder        = func(w io.Writer) Encoder {
		return json.NewEncoder(w)
	}
	XMLEncoder = func(w io.Writer) Encoder {
		return xml.NewEncoder(w)
	}
	TextEncoder = func(w io.Writer) Encoder {
		return &plainTextEncoder{w: w}
	}
	FormEncoder = func(w io.Writer) Encoder {
		return &formEncoder{w: w}
	}
	ByteEncoder = func(w io.Writer) Encoder {
		return &byteEncoder{w: w}
	}
	encoders = map[mime.Type]NewEncoderFunc{
		mime.ApplicationJSON:           JSONEncoder,
		mime.ApplicationXML:            XMLEncoder,
		mime.TextPlain:                 TextEncoder,
		mime.ApplicationFormURLEncoded: FormEncoder,
		mime.ApplicationOctet:          ByteEncoder,
	}
)

// Register takes a mimeType and reginerters a NewEncoderFunc with
func Register(s mime.Type, n NewEncoderFunc) {
	encoders[s] = n
}

func match(specs []header.AcceptSpec, methodSpec []string) []header.AcceptSpec {
	matches := []header.AcceptSpec{}

	methodAccepts := make(map[string]map[string]bool)
	for _, a := range methodSpec {
		group, subgroup := path.Split(a)
		methodAccepts[group][subgroup] = true
	}

	for _, spec := range specs {
		group, subgroup := path.Split(spec.Value)
		if group == "*" {
		}
		if _, ok := methodAccepts[group][subgroup]; ok {
			matches = append(matches, header.AcceptSpec{Q: spec.Q, Value: fmt.Sprintf("%s/%s", group, subgroup)})
		}
	}
	return matches
}

// AcceptsMustMatch checks if the mimetypes for the incoming request <re
// correct.
func AcceptsMustMatch(accepts []mime.Type) endpoint.Middleware {
	return func(next endpoint.Endpoint) endpoint.Endpoint {
		return func(ctx context.Context, request interface{}) (interface{}, error) {
			_, _, err := ResponseEncoder(ctx, accepts)
			if err != nil {
				return nil, err
			}
			return next(ctx, request)
		}
	}
}

// ResponseEncoder returns an encoder and it's corresponding minmetype
func ResponseEncoder(ctx context.Context, accepts []mime.Type) (newEnc NewEncoderFunc, mimeType mime.Type, err error) {
	clientGraph, err := jennyhttp.ContextAccepts(ctx)
	if err != nil {
		return nil, "", errors.Wrap(err, "response encoder")
	}
	serverGraph := mime.Aggregate(accepts)

	available := mime.Intersect(serverGraph, clientGraph)
	if len(available) < 1 { // if nothing intersects
		available = serverGraph // server can do what ever it wants
	}
	err = available.Walk(func(s mime.Type) error {
		if newEnc != nil {
			return nil
		}
		if nef, ok := encoders[s]; ok {
			newEnc = nef
			mimeType = s
			return nil
		}
		return fmt.Errorf("%s isn't a registered encoder", s)
	})

	if err != nil {
		return nil, "", err
	}

	if newEnc == nil || mimeType == "" {
		return encoders[mime.ApplicationOctet], mime.ApplicationOctet, nil
	}
	return newEnc, mimeType, nil
}
