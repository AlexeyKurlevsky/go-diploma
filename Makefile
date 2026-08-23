run:
	go run ./cmd/gophermart/

migrate:
	migrate create -ext sql -dir ./internal/storage/postgres/migrations -seq create_tables