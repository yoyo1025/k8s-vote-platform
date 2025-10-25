gen:
	./tools/scripts/gen-proto.sh

run-auth:
	@cd services/auth && \
	AUTH_PRIVATE_KEY_FILE=./dev_private.pem \
	AUTH_ISSUER=http://localhost:18080 \
	go run ./cmd/auth

run-result-query:
	@cd services/result-query && go run ./cmd/result-query

run-result-api:
	@cd services/result-api && \
	RESULT_QUERY_ADDR=127.0.0.1:50051 \
	go run ./cmd/result-api

run-gateway-sync:
	deck gateway sync --kong-addr http://localhost:8001 ops/kong/kong.yaml

migrate-up:
	GOOSE_DRIVER=postgres GOOSE_DBSTRING="postgres://vote:votepass@localhost:5432/vote?sslmode=disable" goose -dir db/migrations up

start:
	docker compose --profile vote --profile auth --profile result --profile database --profile gateway up --build