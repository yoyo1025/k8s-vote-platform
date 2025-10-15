FROM golang:1.25 AS build

WORKDIR /src
COPY services/result-api/go.mod services/result-api/go.sum ./services/result-api/
COPY gen/go/go.mod gen/go/go.sum ./gen/go/
WORKDIR /src/services/result-api
RUN go mod download

WORKDIR /src
COPY services/result-api ./services/result-api
COPY gen/go ./gen/go
WORKDIR /src/services/result-api
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/result-api ./cmd/result-api

FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=build /out/result-api /usr/local/bin/result-api
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/result-api"]
