FROM golang:1.25-alpine3.22 AS build

WORKDIR /app

COPY . .

RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o worker ./cmd/worker

CMD ["./worker"]