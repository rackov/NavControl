module github.com/rackov/NavControl

go 1.24.6

replace github.com/rackov/NavControlSystem/pkg/models => ./pkg/models

replace github.com/rackov/NavControlSystem/pkg/monitoring => ./pkg/monitoring

require (
	google.golang.org/grpc v1.75.0
	google.golang.org/protobuf v1.36.8
)

require (
	golang.org/x/net v0.41.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.26.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250707201910-8d1bb00bc6a7 // indirect
)
