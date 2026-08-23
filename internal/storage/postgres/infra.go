package storage

import (
	"context"
	"embed"
	"fmt"
	"log"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
)

// Embed the migrations directory into the binary
//
//go:embed migrations/*.sql
var migrationFiles embed.FS

type PostgresStorage struct {
	Pool *pgxpool.Pool
}

// NewPostgresStorage подключается к БД и применяет миграции
func NewPostgresStorage(dsn string) (*PostgresStorage, error) {
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to DB: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping DB: %w", err)
	}
	log.Println("successfuly connect to db")
	defer pool.Close()
	log.Println("Running database migrations...")
	if err := runMigrations(pool); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}
	log.Println("Migrations applied successfully!")

	return &PostgresStorage{Pool: pool}, nil
}

func runMigrations(pool *pgxpool.Pool) error {
	// Grab a standard sql.DB handle from the pgx connection pool config
	db := stdlib.OpenDB(*pool.Config().ConnConfig)
	defer db.Close()

	// Create an iofs driver instance using our embedded files
	sourceDriver, err := iofs.New(migrationFiles, "migrations")
	if err != nil {
		return fmt.Errorf("failed to create source driver: %w", err)
	}

	// Create a pgx database instance for golang-migrate
	targetDriver, err := pgx.WithInstance(db, &pgx.Config{})
	if err != nil {
		return fmt.Errorf("failed to create target database driver: %w", err)
	}

	// Initialize the migrator instance
	m, err := migrate.NewWithInstance(
		"iofs", sourceDriver,
		"pgx", targetDriver,
	)
	if err != nil {
		return fmt.Errorf("failed to initialize migrator: %w", err)
	}

	// Execute up migrations
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return fmt.Errorf("failed to apply up migrations: %w", err)
	}

	return nil
}
