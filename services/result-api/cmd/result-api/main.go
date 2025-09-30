package main

import (
	"log"

	httpapi "github.com/yoyo1025/k8s-vote-platform/services/result-api/internal/http"
)

func main() {
	grpcTarget := "localhost:50051"

	s, err := httpapi.New(grpcTarget)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("result-api HTTP listening on :8080")
	if err := s.Start(":8080"); err != nil {
		log.Fatal(err)
	}
}
