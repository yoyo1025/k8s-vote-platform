package main

import (
	"context"
	"fmt"
	"time"

	resultv1 "github.com/yoyo1025/k8s-vote-platform/gen/go/result/v1"
	"github.com/yoyo1025/k8s-vote-platform/libs/authjwt"
	"google.golang.org/grpc"
)

func main() {
	fmt.Println("devmain:", authjwt.Stub())

	//gRPCクライアントで Ping
	conn, err := grpc.Dial("localhost:50051", grpc.WithInsecure())
	if err != nil {
		panic(err)
	}

	cli := resultv1.NewResultServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	res, err := cli.Ping(ctx, &resultv1.PingRequest{})
	if err != nil {
		panic(err)
	}
	fmt.Println("Ping:", res.GetMessage())
}
