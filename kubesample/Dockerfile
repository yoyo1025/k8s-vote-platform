FROM golang:1.25

WORKDIR /app
COPY ./app /app

RUN go mod download
RUN go build -o app app.go

CMD ["./app"]