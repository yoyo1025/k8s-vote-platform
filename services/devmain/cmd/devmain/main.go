package main

import (
	"fmt"

	resultv1 "github.com/yoyo1025/k8s-vote-platform/gen/go/result/v1"
	"github.com/yoyo1025/k8s-vote-platform/libs/authjwt"
)

func main() {
	fmt.Println("devmain:", authjwt.Stub())
	req := &resultv1.PingRequest{}
	_ = req
	resp := &resultv1.PingResponse{Message: "pong(from generated type)"}
	fmt.Println(resp.GetMessage())
}
