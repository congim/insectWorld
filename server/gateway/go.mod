module insectworld/server/gateway

go 1.26

require (
	github.com/go-sql-driver/mysql v1.8.1
	github.com/gorilla/websocket v1.5.3
	github.com/redis/go-redis/v9 v9.7.0
	github.com/stretchr/testify v1.11.1
	go.uber.org/zap v1.28.0
	golang.org/x/crypto v0.54.0
	google.golang.org/grpc v1.83.0
	insectworld/server/shared v0.0.0
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	github.com/DATA-DOG/go-sqlmock v1.5.2 // indirect
	github.com/alicebob/miniredis/v2 v2.38.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/yuin/gopher-lua v1.1.1 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/net v0.56.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
	golang.org/x/text v0.40.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260526163538-3dc84a4a5aaa // indirect
	google.golang.org/protobuf v1.36.11 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace insectworld/server/shared => ../shared
