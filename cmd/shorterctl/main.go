package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	pb "github.com/jennyservices/shorter/transport/pb"

	"google.golang.org/grpc"
)

func main() {
	var (
		gRPCAddr = flag.String("grpc", ":8081", "gRPC listen address")
	)
	flag.Parse()
	conn, err := grpc.Dial(*gRPCAddr, grpc.WithInsecure(), grpc.WithTimeout(1*time.Second))
	if err != nil {
		log.Fatal(err)
	}
	defer conn.Close()

	client := pb.NewShorterClient(conn)
	resp, err := client.Shorten(context.Background(), &pb.URL{Addr: os.Args[1]})
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(resp.Addr)
}
