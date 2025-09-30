package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
	resultv1 "github.com/yoyo1025/k8s-vote-platform/gen/go/result/v1"
	"google.golang.org/grpc"
)

type Server struct {
	e      *echo.Echo
	client resultv1.ResultServiceClient
}

func New(grpcTarget string) (*Server, error) {
	conn, err := grpc.Dial(grpcTarget, grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("grpc dial: %w", err)
	}
	cli := resultv1.NewResultServiceClient(conn)

	e := echo.New()
	s := &Server{e: e, client: cli}
	s.routes()
	return s, nil
}

func (s *Server) routes() {

	s.e.GET("/healthz", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	// GET /api/v1/results -> gRPC GetTotals を呼んで JSON を返却
	s.e.GET("/api/v1/results", func(c echo.Context) error {
		// deadline をつける
		ctx, cancel := context.WithTimeout(c.Request().Context(), 2*time.Second)
		defer cancel()

		resp, err := s.client.GetTotals(ctx, &resultv1.GetTotalsRequest{})
		if err != nil {
			return c.JSON(http.StatusServiceUnavailable, map[string]any{
				"error": err.Error(),
			})
		}
		return c.JSON(http.StatusOK, resp)
	})

}

func (s *Server) Start(addr string) error {
	return s.e.Start(addr)
}
