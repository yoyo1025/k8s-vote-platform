FROM golang:1.25-alpine3.22 AS build

WORKDIR /app/services/vote-api

COPY services/vote-api/go.mod services/vote-api/go.sum ./
RUN go mod download

COPY services/vote-api ./

RUN CGO_ENABLED=0 GOOS=linux go build -o vote-api ./cmd/vote-api

CMD ["./vote-api"]
