package main

import (
	"log"
	"net"

	resultv1 "github.com/yoyo1025/k8s-vote-platform/gen/go/result/v1"
	"github.com/yoyo1025/k8s-vote-platform/services/result-query/internal/server"
	"google.golang.org/grpc"
)

func main() {
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatal(err)
	}

	s := grpc.NewServer()
	resultv1.RegisterResultServiceServer(s, server.New())

	log.Println("result-query gRPC listening on :50051")
	if err := s.Serve(lis); err != nil {
		log.Fatal(err)
	}
}
