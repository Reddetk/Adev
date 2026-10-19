package repository

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Reddetk/Adev/internal/domain"
)

// The pgx settings store is checked against a real database: the memory double must
// not be the only thing proving the port. Gated by ADEV_TEST_DATABASE_URL.
func openAgentConfigStore(t *testing.T) (*PostgresAgentConfigStore, context.Context) {
	t.Helper()
	dsn := testDatabaseURL()
	if dsn == "" {
		t.Skip("set ADEV_TEST_DATABASE_URL to a dedicated test database to run the postgres store test")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatalf("open pool: %v", err)
	}
	t.Cleanup(pool.Close)
	return NewPostgresAgentConfigStore(pool), ctx
}

func seedTargetSystem(t *testing.T, store *PostgresAgentConfigStore, ctx context.Context, name string) int64 {
	t.Helper()
	var id int64
	err := store.pool.QueryRow(ctx,
		`INSERT INTO target_systems (name, repo_url, description) VALUES ($1, $2, $3) RETURNING id`,
		name, "github.com/example/"+name, "test target system").Scan(&id)
	if err != nil {
		t.Fatalf("seed target system: %v", err)
	}
	t.Cleanup(func() {
		// ON DELETE CASCADE removes the agent configs with it.
		_, _ = store.pool.Exec(context.Background(), `DELETE FROM target_systems WHERE id = $1`, id)
	})
	return id
}

func TestPostgresAgentConfigStoreRoundTrip(t *testing.T) {
	store, ctx := openAgentConfigStore(t)
	id := seedTargetSystem(t, store, ctx, "settings-round-trip")

	if err := store.SetWorkspace(ctx, id, `D:\work\library`); err != nil {
		t.Fatalf("set workspace: %v", err)
	}
	workspace, err := store.Workspace(ctx, id)
	if err != nil {
		t.Fatalf("workspace: %v", err)
	}
	if workspace != `D:\work\library` {
		t.Fatalf("unexpected workspace %q", workspace)
	}

	architect := domain.AgentConfig{
		TargetSystemID: id, Role: domain.RoleArchitect, Provider: domain.ProviderOpenRouter,
		Model: "qwen/qwen3-235b", APIKey: "sk-secret",
	}
	analyst := domain.AgentConfig{
		TargetSystemID: id, Role: domain.RoleAnalyst, Provider: domain.ProviderOllama,
		BaseURL: "http://localhost:11434", Model: "qwen3-coder:30b",
	}
	for _, config := range []domain.AgentConfig{architect, analyst} {
		if err := store.UpsertAgent(ctx, config); err != nil {
			t.Fatalf("upsert %s: %v", config.Role, err)
		}
	}

	agents, err := store.Agents(ctx, id)
	if err != nil {
		t.Fatalf("agents: %v", err)
	}
	if len(agents) != 2 || agents[0].Role != domain.RoleAnalyst || agents[1].Role != domain.RoleArchitect {
		t.Fatalf("unexpected agents: %+v", agents)
	}
	if agents[1].APIKey != "sk-secret" {
		t.Fatal("the store lost the api key")
	}

	single, err := store.Agent(ctx, id, domain.RoleAnalyst)
	if err != nil {
		t.Fatalf("agent: %v", err)
	}
	if single.Model != "qwen3-coder:30b" {
		t.Fatalf("unexpected agent: %+v", single)
	}
}

// Without the foreign key this insert silently created an orphan row.
func TestPostgresAgentConfigStoreRejectsAnUnknownTargetSystem(t *testing.T) {
	store, ctx := openAgentConfigStore(t)

	err := store.UpsertAgent(ctx, domain.AgentConfig{
		TargetSystemID: 999999, Role: domain.RoleAnalyst, Provider: domain.ProviderOllama,
		BaseURL: "http://localhost:11434", Model: "m",
	})

	if !errors.Is(err, domain.ErrSettingNotFound) {
		t.Fatalf("want ErrSettingNotFound, got %v", err)
	}
}

// A known target system without a workspace yields an empty string, not an error:
// the column is empty, not missing. The plan then reports the workspace as missing.
func TestPostgresAgentConfigStoreReportsAnEmptyWorkspace(t *testing.T) {
	store, ctx := openAgentConfigStore(t)
	id := seedTargetSystem(t, store, ctx, "settings-no-workspace")

	workspace, err := store.Workspace(ctx, id)
	if err != nil {
		t.Fatalf("want no error for an empty workspace, got %v", err)
	}
	if workspace != "" {
		t.Fatalf("want an empty workspace, got %q", workspace)
	}
}

func TestPostgresAgentConfigStoreReportsAnUnknownTargetSystem(t *testing.T) {
	store, ctx := openAgentConfigStore(t)

	if _, err := store.Workspace(ctx, 999999); !errors.Is(err, domain.ErrSettingNotFound) {
		t.Fatalf("want ErrSettingNotFound for an unknown target system, got %v", err)
	}
}
