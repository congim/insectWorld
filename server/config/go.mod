module insectworld/server/config

go 1.26

require (
	go.uber.org/zap v1.28.0
	google.golang.org/grpc v1.83.0
	insectworld/server/shared v0.0.0
)

require (
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/net v0.55.0 // indirect
	golang.org/x/sys v0.45.0 // indirect
	golang.org/x/text v0.37.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260526163538-3dc84a4a5aaa // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

replace insectworld/server/shared => ../shared
