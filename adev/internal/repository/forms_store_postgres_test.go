package repository

import (
	"context"
	"os"
	"testing"
)

// testDatabaseURL returns the DSN of the dedicated test database.
// The tests truncate the forms table, so they must never run against the
// development database that .env points at.
func testDatabaseURL() string {
	return os.Getenv("ADEV_TEST_DATABASE_URL")
}

// TestPostgresFormsStore runs the shared store contract against a real Postgres.
//
//	ADEV_TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5432/adev_test?sslmode=disable" go test ./internal/repository/...
func TestPostgresFormsStore(t *testing.T) {
	dsn := testDatabaseURL()
	if dsn == "" {
		t.Skip("set ADEV_TEST_DATABASE_URL to a dedicated test database to run the postgres store test")
	}

	ctx := context.Background()
	store, err := OpenFormsStore(ctx, dsn)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	applyFormsMigration(t, ctx, store)
	if _, err := store.pool.Exec(ctx, `TRUNCATE forms`); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	runStoreSuite(t, store)
}

func TestPostgresFormsStoreIsAppendOnly(t *testing.T) {
	dsn := testDatabaseURL()
	if dsn == "" {
		t.Skip("set ADEV_TEST_DATABASE_URL to a dedicated test database to run the postgres store test")
	}

	ctx := context.Background()
	store, err := OpenFormsStore(ctx, dsn)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	defer store.Close()

	applyFormsMigration(t, ctx, store)
	if _, err := store.pool.Exec(ctx, `TRUNCATE forms`); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	form := buildForm(t, "FRM-TRIGGER", 1, "PIPE-TRIGGER", nil)
	if err := store.Save(ctx, form); err != nil {
		t.Fatalf("save: %v", err)
	}
	if _, err := store.pool.Exec(ctx, `UPDATE forms SET payload = '{"tampered":true}'::jsonb WHERE form_id = 'FRM-TRIGGER'`); err == nil {
		t.Fatal("want the database to reject a payload update")
	}
}

func applyFormsMigration(t *testing.T, ctx context.Context, store *FormsStore) {
	t.Helper()
	migration, err := os.ReadFile("../../migrations/002_forms.up.sql")
	if err != nil {
		t.Fatalf("read migration: %v", err)
	}
	if _, err := store.pool.Exec(ctx, string(migration)); err != nil {
		t.Fatalf("apply migration: %v", err)
	}
}
