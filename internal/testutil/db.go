package testutil

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"runtime"
	"time"

	"github.com/RiceSafe/rice-safe-backend/internal/platform/database"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestDB holds the container and the pgxpool connection
type TestDB struct {
	Container *postgres.PostgresContainer
	Pool      *pgxpool.Pool
	ConnStr   string
}

// SetupTestDB spins up a Postgres testcontainer, runs migrations, and returns the TestDB
func SetupTestDB(ctx context.Context) (*TestDB, error) {
	dbName := "ricesafe_test"
	dbUser := "tester"
	dbPassword := "password"

	postgresContainer, err := postgres.Run(ctx,
		"postgres:16-alpine",
		postgres.WithDatabase(dbName),
		postgres.WithUsername(dbUser),
		postgres.WithPassword(dbPassword),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).WithStartupTimeout(5*time.Second)),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to start container: %w", err)
	}

	connStr, err := postgresContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		return nil, fmt.Errorf("failed to get connection string: %w", err)
	}

	// Wait a tiny bit extra just to be safe before migrations
	time.Sleep(1 * time.Second)

	// Run Migrations
	_, b, _, _ := runtime.Caller(0)
	basepath := filepath.Dir(b)
	// Go up two levels to get to the root migrations folder
	migrationsPath := filepath.Join(basepath, "..", "..", "migrations")

	m, err := migrate.New(
		fmt.Sprintf("file://%s", migrationsPath),
		connStr,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrate instance: %w", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	// Initialize the global DB pool so our repositories can use it
	database.ConnectDB(connStr)
	if database.DB == nil {
		return nil, fmt.Errorf("failed to set global DB pool")
	}

	return &TestDB{
		Container: postgresContainer,
		Pool:      database.DB,
		ConnStr:   connStr,
	}, nil
}

// Teardown cleanly stops the container and closes the pool
func (tdb *TestDB) Teardown(ctx context.Context) {
	if tdb.Pool != nil {
		tdb.Pool.Close()
	}
	if tdb.Container != nil {
		if err := tdb.Container.Terminate(ctx); err != nil {
			log.Printf("failed to terminate container: %s", err)
		}
	}
}

// TruncateAll clears all tables (useful between individual tests)
func (tdb *TestDB) TruncateAll(ctx context.Context) error {
	tables := []string{
		"likes", "comments", "posts",
		"notifications", "notification_settings",
		"outbreaks", "diagnosis_history", "diseases", "users",
	}

	for _, table := range tables {
		// CASCADE ensures dependencies are wiped out too
		_, err := tdb.Pool.Exec(ctx, fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table))
		if err != nil {
			return fmt.Errorf("failed to truncate table %s: %w", table, err)
		}
	}
	return nil
}
