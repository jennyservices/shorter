package main

import (
	"flag"
	"log"
	"net"
	"net/http"

	"github.com/jennyservices/shorter/shorter"
	pb "github.com/jennyservices/shorter/transport/pb"
	v1 "github.com/jennyservices/shorter/transport/v1"
	"google.golang.org/grpc"
)

func main() {
	var (
		addr     = flag.String("addr", ":8080", "default -addr :8080")
		gRPCAddr = flag.String("grpc", ":8081", "gRPC listen address")
	)
	flag.Parse()
	shorterSvc := shorter.New()

	errChan := make(chan error)

	//execute grpc server
	go startGRPCServer(shorterSvc, *gRPCAddr, errChan)
	go startHTTPServer(shorterSvc, *addr, errChan)

	select {
	case err := <-errChan:
		log.Fatal(err)
	}
}

func startGRPCServer(shorterSvc v1.Shorter, addr string, errChan chan error) {
	shorterGRPCServer := v1.NewShorterGRPCServer(shorterSvc)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		errChan <- err
		return
	}
	gRPCServer := grpc.NewServer()
	pb.RegisterShorterServer(gRPCServer, shorterGRPCServer)
	log.Printf("GRPC server listening at %s\n", addr)

	errChan <- gRPCServer.Serve(listener)
}

func startHTTPServer(shorterSvc v1.Shorter, addr string, errChan chan error) {
	shorterHTTPServer := v1.NewShorterHTTPServer(shorterSvc)
	log.Printf("HTTP server listening at %s\n", addr)
	errChan <- http.ListenAndServe(addr, shorterHTTPServer)
}