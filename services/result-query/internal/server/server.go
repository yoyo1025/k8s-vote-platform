package server

import (
	"context"

	resultv1 "github.com/yoyo1025/k8s-vote-platform/gen/go/result/v1"
)

type ResultServer struct {
	resultv1.UnimplementedResultServiceServer
}

func New() *ResultServer {
	return &ResultServer{}
}

func (s *ResultServer) Ping(ctx context.Context, _ *resultv1.PingRequest) (*resultv1.PingResponse, error) {
	return &resultv1.PingResponse{
		Message: "pong"}, nil
}
