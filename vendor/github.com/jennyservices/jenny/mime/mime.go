// Copyright 2017 Typeform SL. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

// Package mime defines commonly used mime types in Jenny
package mime

import (
	"fmt"
	"net/http"
	"path"
	"sort"
	"strings"

	"github.com/golang/gddo/httputil/header"
)

// Types represents a collection of mimeTypes
type Types map[string]map[string]float64

// Type repreents a mimeType
type Type string

const (
	// TextPlain default mimeType for a lot of things
	TextPlain Type = "text/plain"
	// ApplicationJSON application/json type
	ApplicationJSON Type = "application/json"
	// ApplicationXML application/xml type
	ApplicationXML Type = "application/xml"
	// ApplicationOctet application/octet-stream type, for generic stuff
	ApplicationOctet Type = "application/octet-stream"
	// ApplicationFormURLEncoded for form encoded stuff
	ApplicationFormURLEncoded Type = "application/x-www-form-urlencoded"
)

// todo(sevki): there are probably better ways of parsing this but this works
// No need to preoptimize now

// RequestTypes takes a http.Request and builds a mimeTypes from it.
func RequestTypes(r *http.Request) Types {
	g := make(Types)
	for _, v := range header.ParseAccept(r.Header, "Accept") {
		group, subgroup := path.Split(v.Value)
		group = strings.Trim(group, "/")
		if g[group] == nil {
			g[group] = make(map[string]float64)
		}
		for _, subType := range strings.Split(subgroup, "+") {
			g[group][subType] = v.Q
		}

	}
	return g
}

// NewTypes takes a string, possibly one from a header and builds a grpah
// from it.
func NewTypes(s string) Types {
	r, _ := http.NewRequest(http.MethodGet, "url", nil)
	r.Header.Add("Accept", s)
	return RequestTypes(r)
}

// Aggregate takes a array of mime.Type as strings and builds a mime.Types from
// it elements of the array have to be in <type>/<subType> format. Weights are
// not supported.
func Aggregate(list []Type) Types {
	g := make(Types)
	for _, s := range list {
		group, subgroup := path.Split(string(s))
		group = strings.Trim(group, "/")
		if g[group] == nil {
			g[group] = make(map[string]float64)
		}
		for _, subType := range strings.Split(subgroup, "+") {
			g[group][subType] = 1
		}
	}
	return g
}

const wildcard = "*"

// Intersect takes two mime.Types and returns the intersection of two Types'
//  'A' and 'B' and writes each element thats present in both Typess to a new
//  Types. In the result grpah, weights of the node are derived from Types 'B'
//
//   text/html, text/plain;q=0.9          text/troff, text/html;q=0.9
//	+-TypesA--------------------+        +-TypesB--------------------+
//	|            text           |        |              text         |
//	|              +            |        |               +           |
//	|              |            |        |               |           |
//	| plain <------+----> html  |        | html <--------+---> troff |
//	|   +                   +   |        |  +                   +    |
//	|   |                   |   |        |  |                   |    |
//	|   |                   |   |        |  |                   |    |
//	|   v                   v   |        |  v                   v    |
//	| 0.9                   1.0 |        | 0.9                  1.0  |
//	+---------------------------+        +---------------------------+
//
//	+-result----------------+
//	|       text            |
//	|        +              |
//	|        |              |
//	|        v              |
//	|       html            |
//  |        +              |
//	|        |              |
//	|        v              |
//	|       0.9             |
//	+-----------------------+
func Intersect(a Types, b Types) Types {
	g := make(Types)

	// */* is allowed
	// text/* allowed
	// */foo isn't allowed
	if x, ok := b[wildcard]; ok { // if b has a wildcard group
		if q, ok := x[wildcard]; ok { // and it also has a wildcard subgroup
			a.Walk(func(m Type) error {
				if b[m.Group()] == nil {
					b[m.Group()] = make(map[string]float64)
				}
				b[m.Group()][m.SubGroup()] = q // add everything
				return nil
			})
		}
	}
	for group, subgroups := range a {
		if b[group] == nil {
			continue
		}
		_, wildcarded := b[group][wildcard]
		for subgroup := range subgroups {
			if q, ok := b[group][subgroup]; ok || wildcarded {
				if g[group] == nil {
					g[group] = make(map[string]float64)
				}
				g[group][subgroup] = q

			}
		}
	}
	return g
}

// WalkFunc is used for iterating trough mime.Types
type WalkFunc func(Type) error

type mimeSpec struct {
	typ, subTyp string
	q           float64
}

type specs []mimeSpec

// Walk helps you iterate trough mime.Types in a collection
func (t Types) Walk(wf WalkFunc) error {
	types := specs{}
	for typ, subTyps := range t {
		for subTyp, q := range subTyps {
			types = append(types, mimeSpec{
				typ:    typ,
				subTyp: subTyp,
				q:      q,
			})
		}
	}
	sort.Slice(types, func(i int, j int) bool {
		return (types[i].q > types[j].q) || (strings.Compare(types[i].typ, types[j].typ)+strings.Compare(types[i].subTyp, types[j].subTyp) > 0)
	})
	var e error
	e = nil
	for _, typ := range types {
		if err := wf(Type(fmt.Sprintf("%s/%s", typ.typ, typ.subTyp))); err != nil {
			return err
		}
	}
	return e
}

func (t Type) SubGroup() string {
	_, subgroup := path.Split(string(t))
	return subgroup
}

func (t Type) Group() string {
	group, _ := path.Split(string(t))
	group = strings.Trim(group, "/")
	return group
}
