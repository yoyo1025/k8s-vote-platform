FROM golang:1.25 AS build

ARG TARGETOS=linux
ARG TARGETARCH=amd64

WORKDIR /src
COPY services/devmain/go.mod services/devmain/go.sum ./services/devmain/
COPY libs/authjwt/go.mod ./libs/authjwt/
COPY gen/go/go.mod gen/go/go.sum ./gen/go/
WORKDIR /src/services/devmain
RUN go mod download

COPY services/devmain ./services/devmain
COPY libs/authjwt ./libs/authjwt
COPY gen/go ./gen/go
WORKDIR /src/services/devmain
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -o /out/devmain ./cmd/devmain

FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=build /out/devmain /usr/local/bin/devmain
ENTRYPOINT ["/usr/local/bin/devmain"]
