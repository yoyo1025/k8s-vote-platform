FROM golang:1.25-alpine3.22 AS build

WORKDIR /app/services/worker

COPY services/worker/go.mod services/worker/go.sum ./
RUN go mod download

COPY services/worker ./ 

RUN CGO_ENABLED=0 GOOS=linux go build -o worker ./cmd/worker

CMD ["./worker"]
