FROM golang:1.25 AS build

ARG TARGETOS=linux
ARG TARGETARCH=amd64

WORKDIR /src
COPY services/vote-api/go.mod services/vote-api/go.sum ./services/vote-api/
WORKDIR /src/services/vote-api
RUN go mod download

COPY services/vote-api ./services/vote-api
WORKDIR /src/services/vote-api
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o /out/vote-api ./cmd/vote-api

FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=build /out/vote-api /usr/local/bin/vote-api
EXPOSE 9080
ENTRYPOINT ["/usr/local/bin/vote-api"]
