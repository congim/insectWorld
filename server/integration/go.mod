module insectworld/server/integration

go 1.26

require (
	github.com/stretchr/testify v1.11.1
	go.uber.org/zap v1.28.0
	insectworld/server/combat v0.0.0
	insectworld/server/economy v0.0.0
	insectworld/server/operation v0.0.0
	insectworld/server/shared v0.0.0
	insectworld/server/social v0.0.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	insectworld/server/combat => ../combat
	insectworld/server/economy => ../economy
	insectworld/server/operation => ../operation
	insectworld/server/shared => ../shared
	insectworld/server/social => ../social
)
