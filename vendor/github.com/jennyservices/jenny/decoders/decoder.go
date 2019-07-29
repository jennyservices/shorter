// Copyright 2017 Typeform SL. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package decoders

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"

	"github.com/gorilla/schema"
	"github.com/jennyservices/jenny/mime"
)

var (
	// ErrDecoderNotFound is returned when a reuqest doesn't have
	// enough information to determine a decoder
	ErrDecoderNotFound = errors.New("decoder could not be found")
	// JSONDecoder decodes data from a http.Request
	JSONDecoder = func(r io.Reader) Decoder {
		return json.NewDecoder(r)
	}
	// XMLDecoder decodes data from a http.Request
	XMLDecoder = func(r io.Reader) Decoder {
		return xml.NewDecoder(r)
	}
	// FormDecoder decodes data from a http.Request
	FormDecoder = func(r io.Reader) Decoder {
		return &formDecoder{r: r}
	}
	decoders = map[mime.Type]newDecoder{
		mime.ApplicationJSON:           JSONDecoder,
		mime.ApplicationXML:            XMLDecoder,
		mime.ApplicationFormURLEncoded: FormDecoder,
	}
)

// Register registers a new decoder to be used with jenny endpoints, it is to be
// recalled based on the mime-type
func Register(s mime.Type, d newDecoder) {
	decoders[s] = d
}

// Decoder is an interface that decodes http.Request.Body from their
// Content-Type mime types.
type Decoder interface {
	Decode(v interface{}) error
}

type newDecoder func(io.Reader) Decoder

type formDecoder struct {
	r io.Reader
}

func (f *formDecoder) Decode(i interface{}) error {
	dec := schema.NewDecoder()
	body, err := ioutil.ReadAll(f.r)
	log.Println(string(body))
	if err != nil {
		return fmt.Errorf("decoding form: reading body: %v", err)
	}
	values, err := url.ParseQuery(string(body))
	if err != nil {
		return fmt.Errorf("decoding form: parsing values: %v", err)
	}
	if len(values) <= 0 {
		return fmt.Errorf("decoding form: no values found")
	}
	return dec.Decode(i, values)
}

// ResponseDecoder returns a decoder for a given http.Request
func ResponseDecoder(r *http.Response) (Decoder, error) {
	serverSent := mime.NewTypes(r.Header.Get("Content-Type"))
	var dec Decoder
	err := serverSent.Walk(func(x mime.Type) error {
		if decoderFunc, ok := decoders[x]; ok {
			dec = decoderFunc(r.Body)
			return nil
		}
		return fmt.Errorf("%s isn't a registered decoder", x)
	})
	if dec == nil {
		return nil, err
	} else {
		return dec, nil
	}
}

// RequestDecoder returns a decoder for a given http.Request
func RequestDecoder(r *http.Request, accepts []mime.Type) (Decoder, error) {
	serverAccepts := mime.Aggregate(accepts)
	clientSent := mime.NewTypes(r.Header.Get("Content-Type"))
	available := mime.Intersect(serverAccepts, clientSent)

	if len(available) < 0 {
		available = serverAccepts
	}
	var dec Decoder
	err := available.Walk(func(x mime.Type) error {
		if decoderFunc, ok := decoders[x]; ok {
			dec = decoderFunc(r.Body)
			return nil
		}
		return fmt.Errorf("%s isn't a registered decoder", x)
	})
	if err != nil {
		return nil, err
	}
	if dec == nil {
		return nil, fmt.Errorf("coudln't find decoder for %q", accepts)
	}
	return dec, nil
}
