SERVICES := auth result-api result-query vote-api worker devmain

IMAGE_REGISTRY ?=
IMAGE_NAMESPACE ?= k8s-vote-platform
IMAGE_TAG ?= latest
DOCKER_BUILD_ARGS ?=

image_ref = $(strip $(if $(IMAGE_REGISTRY),$(IMAGE_REGISTRY)/,))$(IMAGE_NAMESPACE)/$1:$(IMAGE_TAG)

.PHONY: gen
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

.PHONY: docker-build docker-build-all $(SERVICES:%=docker-build-%)
docker-build:
ifdef SERVICE
	$(MAKE) docker-build-$(SERVICE)
else
	$(MAKE) docker-build-all
endif

docker-build-all: $(SERVICES:%=docker-build-%)

docker-build-%:
	docker build $(DOCKER_BUILD_ARGS) \
		-f services/$*/Dockerfile \
		-t $(call image_ref,$*) \
		.
