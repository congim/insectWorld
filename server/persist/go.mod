module insectworld/server/persist

go 1.26

require (
	github.com/go-sql-driver/mysql v1.8.1
	go.uber.org/zap v1.28.0
	insectworld/server/shared v0.0.0
)

require (
	filippo.io/edwards25519 v1.1.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
)

replace insectworld/server/shared => ../shared
