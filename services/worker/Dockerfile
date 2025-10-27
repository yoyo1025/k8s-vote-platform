FROM golang:1.25 AS build

WORKINGDIR /app
COPY . .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o worker ./services/worker

COPY --from=build /out/worker /usr/local/bin/worker
CMD ["/usr/local/bin/worker"]