package httpapi

import (
	"context"
	"encoding/json"
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
			return c.JSON(http.StatusServiceUnavailable, map[string]any{"error": err.Error()})
		}
		return c.JSON(http.StatusOK, resp)
	})

	s.e.GET("/api/v1/results/stream", func(c echo.Context) error {
		ctx, cancel := context.WithTimeout(c.Request().Context(), 10*time.Minute)
		defer cancel()

		stream, err := s.client.SubscribeTotals(ctx, &resultv1.SubscribeTotalsRequest{Tenant: "default"})
		if err != nil {
			return c.JSON(http.StatusServiceUnavailable, map[string]any{"error": err.Error()})
		}
		// SSE ヘッダ
		c.Response().Header().Set(echo.HeaderContentType, "text/event-stream")
		c.Response().Header().Set("Cache-Control", "no-cache")
		c.Response().Header().Set("Connection", "keep-alive")
		c.Response().WriteHeader(http.StatusOK)
		flusher, ok := c.Response().Writer.(http.Flusher)
		if !ok {
			return c.String(http.StatusInternalServerError, "streaming unsupported")
		}

		// 1分ごとにハートビート送信
		heartbeat := time.NewTicker(1 * time.Minute)
		defer heartbeat.Stop()

		// 初回コメント
		_, _ = c.Response().Writer.Write([]byte(":ok\n\n"))
		flusher.Flush()

		// 受信ループ：gRPC → JSON → SSE data
		for {
			select {
			case <-c.Request().Context().Done():
				// ブラウザ切断。gRPC の ctx もキャンセルされる
				return nil
			case <-heartbeat.C:
				// 心拍（コメント行）
				_, _ = c.Response().Writer.Write([]byte(":ping\n\n"))
				flusher.Flush()
			default:
				// ノンブロッキングだと忙しいので、短い受信待ちでブロック
				msg, err := stream.Recv()
				if err != nil {
					// 下流切断：ここで終了（ブラウザ側は EventSource が再接続する）
					// ここで 204 を返す必要はない（既にヘッダ送信済み）
					return nil
				}
				b, _ := json.Marshal(msg) // 小さく行く：そのまま JSON に
				// SSE フレーム：data: <json>\n\n
				_, _ = c.Response().Writer.Write([]byte("data: "))
				_, _ = c.Response().Writer.Write(b)
				_, _ = c.Response().Writer.Write([]byte("\n\n"))
				flusher.Flush()
			}
		}
	})
}

func (s *Server) Start(addr string) error {
	return s.e.Start(addr)
}
