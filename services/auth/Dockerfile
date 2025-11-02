FROM golang:1.25-alpine3.22 AS build

WORKDIR /app/services/auth

COPY services/auth/go.mod services/auth/go.sum ./
RUN go mod download

COPY services/auth ./ 

RUN CGO_ENABLED=0 GOOS=linux go build -o auth ./cmd/auth

CMD ["./auth"]
