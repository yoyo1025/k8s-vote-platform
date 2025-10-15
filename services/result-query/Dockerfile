FROM golang:1.25 AS build

WORKDIR /src
COPY services/result-query/go.mod services/result-query/go.sum ./services/result-query/
COPY gen/go/go.mod gen/go/go.sum ./gen/go/
WORKDIR /src/services/result-query
RUN go mod download

WORKDIR /src
COPY services/result-query ./services/result-query
COPY gen/go ./gen/go
WORKDIR /src/services/result-query
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /out/result-query ./cmd/result-query

FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=build /out/result-query /usr/local/bin/result-query
EXPOSE 50051
ENTRYPOINT ["/usr/local/bin/result-query"]
