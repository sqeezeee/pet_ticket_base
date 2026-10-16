//go:build integration

package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	app "pet-ticket/internal/app/tickets"
	domain "pet-ticket/internal/domain/tickets"
	"pet-ticket/internal/infra/migration"
)

func setupLocalTestDB(t testing.TB, dsn string) *testDB {
	t.Helper()
	admin, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	var name string
	if err := admin.QueryRow("SELECT current_database()").Scan(&name); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(name, "pet_ticket_test_") {
		t.Fatal("PET_TICKET_TEST_DSN must use a dedicated database named pet_ticket_test_*")
	}
	schema := fmt.Sprintf("test_%d", time.Now().UnixNano())
	if _, err := admin.Exec("CREATE SCHEMA " + schema); err != nil {
		t.Fatal(err)
	}
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		u, err := url.Parse(dsn)
		if err != nil {
			t.Fatal(err)
		}
		q := u.Query()
		q.Set("search_path", schema)
		u.RawQuery = q.Encode()
		dsn = u.String()
	} else {
		dsn += " search_path=" + schema
	}
	conn, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	if err := migration.Up(conn); err != nil {
		t.Fatal(err)
	}
	return &testDB{db: &DB{conn: conn}, connStr: dsn}
}

// This crosses the actual app -> postgres boundary; repository-only tests missed the old typed-key mismatch.
func TestServiceHistoryFailureRollsBack_Integration(t *testing.T) {
	db := setupTestDB(t).db
	repo := NewTicketsRepository(db)
	svc := app.NewService(repo, db, zerolog.Nop())
	ctx := context.Background()
	created, err := svc.CreateTicket(ctx, app.CreateTicketInput{UserID: 1, TopicID: 1, Comment: "original"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.conn.Exec("ALTER TABLE ticket_history ADD CONSTRAINT reject_changes CHECK (action = 'created')"); err != nil {
		t.Fatal(err)
	}
	comment := "must rollback"
	if _, err := svc.UpdateTicket(ctx, app.UpdateTicketInput{ID: created.ID, Comment: &comment}); err == nil {
		t.Fatal("expected history rejection")
	}
	got, err := repo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Comment != "original" {
		t.Fatalf("update escaped rollback: %q", got.Comment)
	}
	// Reject creation's history too; the new ticket must not survive.
	if _, err := db.conn.Exec("ALTER TABLE ticket_history ADD CONSTRAINT reject_new CHECK (user_id <> 2)"); err != nil {
		t.Fatal(err)
	}
	if _, err := svc.CreateTicket(ctx, app.CreateTicketInput{UserID: 2, TopicID: 1, Comment: "rollback create"}); err == nil {
		t.Fatal("expected create rejection")
	}
	var count int
	if err := db.conn.QueryRow("SELECT count(*) FROM tickets WHERE user_id=2").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("create escaped rollback: %d", count)
	}
}

func TestServiceConcurrentUpdatesPreserveBothFields_Integration(t *testing.T) {
	db := setupTestDB(t).db
	repo := NewTicketsRepository(db)
	svc := app.NewService(repo, db, zerolog.Nop())
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	created, err := svc.CreateTicket(ctx, app.CreateTicketInput{UserID: 1, TopicID: 1, Comment: "original"})
	if err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	status := domain.StatusInProgress
	comment := "updated"
	for _, input := range []app.UpdateTicketInput{{ID: created.ID, Status: &status}, {ID: created.ID, Comment: &comment}} {
		wg.Add(1)
		go func(in app.UpdateTicketInput) {
			defer wg.Done()
			<-start
			_, err := svc.UpdateTicket(ctx, in)
			errs <- err
		}(input)
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	got, err := repo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != status || got.Comment != comment {
		t.Fatalf("lost update: %+v", got)
	}
	history, err := repo.GetHistory(ctx, created.ID, 10, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(history) != 3 {
		t.Fatalf("want created + two changes, got %d", len(history))
	}
}

func TestMigrationParityAndRollback_Integration(t *testing.T) {
	db := setupTestDB(t).db
	if err := migration.Up(db.conn); err != nil {
		t.Fatal(err)
	} // same runner, no-op second time
	if err := migration.Down(db.conn); err != nil {
		t.Fatal(err)
	}
	if err := migration.Up(db.conn); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := db.conn.QueryRow("SELECT count(*) FROM ticket_statuses").Scan(&count); err != nil || count != 5 {
		t.Fatalf("seed after replay: %d %v", count, err)
	}
}

func TestLegacyGooseDatabaseRefused_Integration(t *testing.T) {
	db := setupTestDB(t).db
	if _, err := db.conn.Exec("CREATE TABLE goose_db_version (id bigint)"); err != nil {
		t.Fatal(err)
	}
	if err := migration.Up(db.conn); err == nil || !strings.Contains(err.Error(), "legacy Goose") {
		t.Fatalf("expected explicit legacy refusal, got %v", err)
	}
	var count int
	if err := db.conn.QueryRow("SELECT count(*) FROM ticket_statuses").Scan(&count); err != nil || count != 5 {
		t.Fatalf("legacy guard altered schema: %d %v", count, err)
	}
}

func TestStatusSeedMatchesDomain_Integration(t *testing.T) {
	db := setupTestDB(t).db
	repo := NewTicketsRepository(db)
	statuses, err := repo.GetAllStatuses(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(statuses) != 5 {
		t.Fatalf("expected 5 statuses, got %d", len(statuses))
	}
	for _, s := range statuses {
		if domain.Status(s.ID).String() != s.Name {
			t.Fatalf("status contract mismatch: %+v", s)
		}
	}
}
