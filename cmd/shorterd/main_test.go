package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	pb "github.com/jennyservices/shorter/transport/pb"

	v1 "github.com/jennyservices/shorter/transport/v1"
	"github.com/phayes/freeport"
	"google.golang.org/grpc"
)

//  e2e Tests :)
type mockShorter struct {
	shorten func(ctx context.Context, Long v1.URL) (Body *v1.URL, err error)
}

func (s *mockShorter) Shorten(ctx context.Context, Long v1.URL) (Body *v1.URL, err error) {
	return s.shorten(ctx, Long)
}

const (
	request  = "hello"
	response = "goodbye"
)

// This test here is to test only that the gRPC server is working and nothing more.
func TestGRPC(t *testing.T) {
	errChan := make(chan error)
	port, err := freeport.GetFreePort()
	if err != nil {
		log.Fatal(err)
	}

	shortenFunc := func(ctx context.Context, long v1.URL) (Body *v1.URL, err error) {
		if long.Addr != request {
			return nil, errors.New("whooops")
		}

		return &v1.URL{Addr: response}, nil
	}

	grpcAddr := fmt.Sprintf(":%d", port)
	// port is ready to listen on
	go startGRPCServer(&mockShorter{shorten: shortenFunc}, grpcAddr, errChan)

	conn, err := grpc.Dial(grpcAddr, grpc.WithInsecure(), grpc.WithTimeout(1*time.Second))
	defer conn.Close()

	client := pb.NewShorterClient(conn)
	resp, err := client.Shorten(context.Background(), &pb.URL{Addr: request})
	if resp.Addr != response {
		t.Fail()
	}
}

func TestHTTPWorks(t *testing.T) {
	shortenFunc := func(ctx context.Context, long v1.URL) (Body *v1.URL, err error) {
		if long.Addr != request {
			return nil, errors.New("whooops")
		}

		return &v1.URL{Addr: response}, nil
	}
	shorterHTTPServer := v1.NewShorterHTTPServer(&mockShorter{shorten: shortenFunc})
	ts := httptest.NewServer(shorterHTTPServer)
	defer ts.Close()

	log.Println(ts.URL)

	buf := bytes.NewBufferString(`{"addr": "hello"}`)

	req, err := http.NewRequest(http.MethodPost, ts.URL+"/shorten", buf)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	defer resp.Body.Close()
	dec := json.NewDecoder(resp.Body)

	u := v1.URL{}
	dec.Decode(&u)
	if u.Addr != response {
		t.FailNow()
	}
}
func TestHTTPWrongContentType(t *testing.T) {
	shortenFunc := func(ctx context.Context, long v1.URL) (Body *v1.URL, err error) {
		if long.Addr != request {
			return nil, errors.New("whooops")
		}

		return &v1.URL{Addr: response}, nil
	}
	shorterHTTPServer := v1.NewShorterHTTPServer(&mockShorter{shorten: shortenFunc})
	ts := httptest.NewServer(shorterHTTPServer)
	defer ts.Close()

	buf := bytes.NewBufferString(`{"addr": "hello"}`)

	req, err := http.NewRequest(http.MethodPost, ts.URL+"/shorten", buf)
	if err != nil {
		t.Log(err)
		t.FailNow()
	}

	req.Header.Set("Content-Type", "application/xml")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.FailNow()
	}

	defer resp.Body.Close()
	dec := json.NewDecoder(resp.Body)

	u := v1.URL{}

	if err := dec.Decode(&u); err == nil {
		t.Log("this should have thrown an error")
		t.FailNow()
	}
}
