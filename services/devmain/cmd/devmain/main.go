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
	defer conn.Close()

	cli := resultv1.NewResultServiceClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	res, err := cli.Ping(ctx, &resultv1.PingRequest{})
	if err != nil {
		panic(err)
	}
	fmt.Println("Ping:", res.GetMessage())

	{
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		res, err := cli.GetTotals(ctx, &resultv1.GetTotalsRequest{})
		if err != nil {
			panic(err)
		}
		fmt.Println("GetTotals:", res.Totals, res.GetUpdatedAt())
	}

	{
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()
		stream, err := cli.SubscribeTotals(ctx, &resultv1.SubscribeTotalsRequest{Tenant: "default"})
		if err != nil {
			panic(err)
		}

		for {
			msg, err := stream.Recv()
			if err != nil {
				fmt.Println("stream end:", err)
				break
			}
			fmt.Println("SubscribeTotals:", msg.Totals, msg.GetUpdatedAt())
		}
	}
}
