package testutil_test

import (
	"context"
	"testing"

	"github.com/RiceSafe/rice-safe-backend/internal/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupTestDB(t *testing.T) {
	ctx := context.Background()

	// Spin it up
	db, err := testutil.SetupTestDB(ctx)
	require.NoError(t, err)
	require.NotNil(t, db)

	// Clean up after the test finishes
	defer db.Teardown(ctx)

	// Verify the connection pool works
	err = db.Pool.Ping(ctx)
	assert.NoError(t, err, "Database pool should be pingable")

	// Verify migrations ran (tables should exist)
	var count int
	err = db.Pool.QueryRow(ctx, "SELECT count(*) FROM users").Scan(&count)
	assert.NoError(t, err, "Querying the users table should succeed, confirming migrations ran")
	assert.Equal(t, 0, count, "Users table should be empty initially")
}
