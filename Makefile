DB_USER ?= postgres
DB_NAME ?= block_explorer

.PHONY: reset-dev-data reset-dev-schema test

reset-dev-data:
	psql -U $(DB_USER) -d $(DB_NAME) -c "TRUNCATE TABLE transactions, blocks RESTART IDENTITY CASCADE;"

reset-dev-schema:
	psql -U $(DB_USER) -d $(DB_NAME) -c "DROP TABLE IF EXISTS transactions CASCADE; DROP TABLE IF EXISTS blocks CASCADE;"

test:
	go test ./...