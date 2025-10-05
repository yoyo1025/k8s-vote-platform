package main

import (
	"log"

	"github.com/yoyo1025/k8s-vote-platform/services/auth/internal/auth"
)

func main() {
	s, err := auth.New()
	if err != nil {
		log.Fatal(err)
	}

	// 開発は :18080 で待受
	log.Println("auth HTTP listening on :18080")
	if err := s.Start(":18080"); err != nil {
		log.Fatal(err)
	}
}
