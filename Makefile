run:
	go run ./cmd/gophermart/

migrate:
	migrate create -ext sql -dir ./internal/storage/postgres/migrations -seq create_tables

generate_mock:
	mockgen -source=internal/storage/repository.go -destination=internal/storage/mocks/mocks.go -package=mocks