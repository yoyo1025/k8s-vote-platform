package server

import (
	"context"
	"time"

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
		Message: "pong",
	}, nil
}

func (s *ResultServer) GetTotals(ctx context.Context, _ *resultv1.GetTotalsRequest) (*resultv1.GetTotalsResponse, error) {
	return &resultv1.GetTotalsResponse{
		Totals: []*resultv1.Totals{
			{CandidateId: 1, Count: 42},
			{CandidateId: 2, Count: 17},
		},
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}, nil
}

func (s *ResultServer) SubscribeTotals(req *resultv1.SubscribeTotalsRequest, stream resultv1.ResultService_SubscribeTotalsServer) error {
	for i := 0; i < 10; i++ {
		resp := &resultv1.SubscribeTotalsResponse{
			Totals: []*resultv1.Totals{
				{CandidateId: 1, Count: 42 + uint64(i)},
				{CandidateId: 2, Count: 17 + uint64(i)},
			},
			UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		}
		if err := stream.Send(resp); err != nil {
			return err
		}
		time.Sleep(1 * time.Second)
	}
	return nil
}
