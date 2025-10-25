FROM golang:1.25 AS build

ARG TARGETOS=linux
ARG TARGETARCH=amd64

WORKDIR /src
COPY services/auth/go.mod services/auth/go.sum ./services/auth/
WORKDIR /src/services/auth
RUN go mod download

COPY services/auth ./
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o /out/auth ./cmd/auth

FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=build /out/auth /usr/local/bin/auth
COPY --from=build /src/services/auth/dev_private.pem /app/dev_private.pem
ENV AUTH_PRIVATE_KEY_FILE=/app/dev_private.pem
EXPOSE 18080
ENTRYPOINT ["/usr/local/bin/auth"]
