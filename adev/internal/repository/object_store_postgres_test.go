package repository

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Reddetk/Adev/internal/domain"
	"github.com/Reddetk/Adev/internal/usecase"
)

// TestPostgresObjectStore runs the registry store against a real Postgres.
//
//	ADEV_TEST_DATABASE_URL="postgres://postgres:postgres@localhost:5432/adev_test?sslmode=disable" go test ./internal/repository/...
func TestPostgresObjectStore(t *testing.T) {
	dsn := testDatabaseURL()
	if dsn == "" {
		t.Skip("set ADEV_TEST_DATABASE_URL to a dedicated test database to run the postgres object store test")
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("open pool: %v", err)
	}
	defer pool.Close()

	if _, err := pool.Exec(ctx, `TRUNCATE object_entries`); err != nil {
		t.Fatalf("truncate: %v", err)
	}

	var objectStore usecase.ObjectStore = NewPostgresObjectStore(pool)
	introducedBy := domain.Ref{FormID: "FRM-1-03", FormType: domain.TypeHLD, FormVersion: 1}
	object := domain.Object{
		ObjectID:       "OBJ-container-001",
		TargetSystemID: 1,
		Kind:           domain.KindContainer,
		Name:           "obj-container-1",
		Status:         domain.ObjectPlanned,
		Fields:         map[string]any{"tech": "go"},
		IntroducedBy:   introducedBy,
		ChangedBy:      []domain.Ref{},
	}
	if err := objectStore.Save(ctx, object); err != nil {
		t.Fatalf("save: %v", err)
	}

	read, err := objectStore.Get(ctx, "OBJ-container-001")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if read.Name != "obj-container-1" || read.Fields["tech"] != "go" || read.Status != domain.ObjectPlanned {
		t.Fatalf("bad object: %+v", read)
	}
	if read.IntroducedBy.FormID != "FRM-1-03" {
		t.Fatalf("bad introducedBy: %+v", read.IntroducedBy)
	}

	next, err := objectStore.Allocate(ctx, domain.KindContainer)
	if err != nil {
		t.Fatalf("allocate: %v", err)
	}
	if next != 2 {
		t.Fatalf("allocate must continue the kind sequence, got %d", next)
	}

	// Save upserts: one transition keeps one entry per object id.
	read.Status = domain.ObjectActive
	if err := objectStore.Save(ctx, read); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	objects, err := objectStore.FindByTargetSystem(ctx, 1)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(objects) != 1 || objects[0].Status != domain.ObjectActive {
		t.Fatalf("the upsert must keep one entry: %+v", objects)
	}

	if err := objectStore.Save(ctx, domain.Object{ObjectID: "OBJ-container-002", TargetSystemID: 2, Kind: domain.KindContainer, Name: "obj-container-2", Status: domain.ObjectPlanned, IntroducedBy: introducedBy, ChangedBy: []domain.Ref{}}); err != nil {
		t.Fatalf("save other: %v", err)
	}
	objects, err = objectStore.FindByTargetSystem(ctx, 1)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(objects) != 1 {
		t.Fatalf("the list of a target system must not leak another: %+v", objects)
	}
}
