module github.com/yoyo1025/k8s-vote-platform/services/result-api

replace github.com/yoyo1025/k8s-vote-platform/gen/go => ../../gen/go

go 1.25.1

require (
	github.com/labstack/echo/v4 v4.13.4
	github.com/yoyo1025/k8s-vote-platform/gen/go v0.0.0-00010101000000-000000000000
)

require (
	github.com/labstack/gommon v0.4.2 // indirect
	github.com/mattn/go-colorable v0.1.14 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/valyala/bytebufferpool v1.0.0 // indirect
	github.com/valyala/fasttemplate v1.2.2 // indirect
	golang.org/x/crypto v0.39.0 // indirect
	golang.org/x/net v0.41.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.26.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250707201910-8d1bb00bc6a7 // indirect
	google.golang.org/grpc v1.75.1 // indirect
	google.golang.org/protobuf v1.36.9 // indirect
)
