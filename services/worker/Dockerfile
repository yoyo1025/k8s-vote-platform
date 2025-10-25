FROM golang:1.25 AS build

ARG TARGETOS=linux
ARG TARGETARCH=amd64

WORKDIR /src
COPY services/worker/go.mod services/worker/go.sum ./services/worker/
WORKDIR /src/services/worker
RUN go mod download

COPY services/worker ./services/worker
WORKDIR /src/services/worker
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o /out/worker ./cmd/worker

FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=build /out/worker /usr/local/bin/worker
ENTRYPOINT ["/usr/local/bin/worker"]
