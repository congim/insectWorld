module insectworld/tests/integration

go 1.26

require (
	github.com/go-sql-driver/mysql v1.10.0
	github.com/stretchr/testify v1.11.1
	go.uber.org/zap v1.28.0
	insectworld/server/combat v0.0.0
	insectworld/server/economy v0.0.0
	insectworld/server/game v0.0.0
	insectworld/server/gateway v0.0.0
	insectworld/server/operation v0.0.0
	insectworld/server/shared v0.0.0
	insectworld/server/social v0.0.0
)

require (
	filippo.io/edwards25519 v1.2.0 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	insectworld/server/combat => ../../server/combat
	insectworld/server/economy => ../../server/economy
	insectworld/server/game => ../../server/game
	insectworld/server/gateway => ../../server/gateway
	insectworld/server/operation => ../../server/operation
	insectworld/server/shared => ../../server/shared
	insectworld/server/social => ../../server/social
)
