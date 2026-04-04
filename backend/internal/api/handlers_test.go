package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hiro1202/fourkeys-insights/internal/db"
	"go.uber.org/zap"
)

func setupTestHandler(t *testing.T) (*Handler, db.Store) {
	t.Helper()
	store, err := db.NewSQLiteStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { store.Close() })

	logger, _ := zap.NewDevelopment()
	h := &Handler{Store: store, Logger: logger}
	return h, store
}

func TestListGroups_Empty(t *testing.T) {
	h, _ := setupTestHandler(t)
	router := NewRouter(h, h.Logger)

	req := httptest.NewRequest("GET", "/api/v1/groups", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var groups []interface{}
	json.NewDecoder(w.Body).Decode(&groups)
	if len(groups) != 0 {
		t.Fatalf("expected empty list, got %d", len(groups))
	}
}

func TestCreateAndListGroups(t *testing.T) {
	h, store := setupTestHandler(t)
	router := NewRouter(h, h.Logger)

	r1, _ := store.UpsertRepo(context.Background(), &db.Repo{Owner: "o", Name: "a", FullName: "o/a", DefaultBranch: "main"})
	r2, _ := store.UpsertRepo(context.Background(), &db.Repo{Owner: "o", Name: "b", FullName: "o/b", DefaultBranch: "main"})

	body, _ := json.Marshal(createGroupRequest{
		Name:            "backend",
		AggregationUnit: "weekly",
		RepoIDs:         []int64{r1, r2},
	})
	req := httptest.NewRequest("POST", "/api/v1/groups", bytes.NewReader(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	var group db.Group
	json.NewDecoder(w.Body).Decode(&group)
	if group.Name != "backend" {
		t.Fatalf("expected name 'backend', got '%s'", group.Name)
	}
	if len(group.Repos) != 2 {
		t.Fatalf("expected 2 repos, got %d", len(group.Repos))
	}

	req = httptest.NewRequest("GET", "/api/v1/groups", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var groups []db.Group
	json.NewDecoder(w.Body).Decode(&groups)
	if len(groups) != 1 {
		t.Fatalf("expected 1 group, got %d", len(groups))
	}
}

func TestCreateGroup_Validation(t *testing.T) {
	h, _ := setupTestHandler(t)
	router := NewRouter(h, h.Logger)

	// Missing name
	body, _ := json.Marshal(createGroupRequest{RepoIDs: []int64{1}})
	req := httptest.NewRequest("POST", "/api/v1/groups", bytes.NewReader(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing name, got %d", w.Code)
	}

	// Missing repos
	body, _ = json.Marshal(createGroupRequest{Name: "test"})
	req = httptest.NewRequest("POST", "/api/v1/groups", bytes.NewReader(body))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing repos, got %d", w.Code)
	}
}

func TestUpdateGroupSettingsAndDelete(t *testing.T) {
	h, store := setupTestHandler(t)
	router := NewRouter(h, h.Logger)

	r1, _ := store.UpsertRepo(context.Background(), &db.Repo{Owner: "o", Name: "a", FullName: "o/a", DefaultBranch: "main"})
	store.CreateGroup(context.Background(), "team", "weekly", []int64{r1})

	// Update settings
	body, _ := json.Marshal(updateGroupSettingsRequest{
		Name:            "new-team",
		AggregationUnit: "monthly",
		LeadTimeStart:   "pr_created_at",
		MTTRStart:       "issue.created_at",
	})
	req := httptest.NewRequest("PUT", "/api/v1/groups/1/settings", bytes.NewReader(body))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	// Verify settings
	req = httptest.NewRequest("GET", "/api/v1/groups/1/settings", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var settings map[string]interface{}
	json.NewDecoder(w.Body).Decode(&settings)
	if settings["aggregation_unit"] != "monthly" {
		t.Fatalf("expected aggregation_unit 'monthly', got '%v'", settings["aggregation_unit"])
	}
	if settings["mttr_start"] != "issue.created_at" {
		t.Fatalf("expected mttr_start 'issue.created_at', got '%v'", settings["mttr_start"])
	}

	// Delete
	req = httptest.NewRequest("DELETE", "/api/v1/groups/1", nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}
}

func TestRepoSettings(t *testing.T) {
	h, store := setupTestHandler(t)
	router := NewRouter(h, h.Logger)

	store.UpsertRepo(context.Background(), &db.Repo{Owner: "o", Name: "r", FullName: "o/r", DefaultBranch: "main"})

	req := httptest.NewRequest("GET", "/api/v1/repos/1/settings", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var settings db.RepoSettings
	json.NewDecoder(w.Body).Decode(&settings)
	if settings.LeadTimeStart != "first_commit_at" {
		t.Fatalf("expected default lead_time_start 'first_commit_at', got '%s'", settings.LeadTimeStart)
	}

	// Update settings
	body, _ := json.Marshal(db.RepoSettings{
		IncidentRules: `{"title_keywords":["rollback"]}`,
		LeadTimeStart: "pr_created_at",
		MTTRStart:     "issue.created_at",
	})
	req = httptest.NewRequest("PUT", "/api/v1/repos/1/settings", bytes.NewReader(body))
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestBadge(t *testing.T) {
	h, store := setupTestHandler(t)
	router := NewRouter(h, h.Logger)

	r1, _ := store.UpsertRepo(context.Background(), &db.Repo{Owner: "o", Name: "r", FullName: "o/r", DefaultBranch: "main"})
	store.CreateGroup(context.Background(), "team", "weekly", []int64{r1})

	req := httptest.NewRequest("GET", "/api/v1/groups/1/badge", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "image/svg+xml" {
		t.Fatalf("expected SVG content type, got '%s'", ct)
	}
}

func TestListGroupPulls_Empty(t *testing.T) {
	h, store := setupTestHandler(t)
	router := NewRouter(h, h.Logger)

	r1, _ := store.UpsertRepo(context.Background(), &db.Repo{Owner: "o", Name: "r", FullName: "o/r", DefaultBranch: "main"})
	store.CreateGroup(context.Background(), "team", "weekly", []int64{r1})

	req := httptest.NewRequest("GET", "/api/v1/groups/1/pulls", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var result map[string]interface{}
	json.NewDecoder(w.Body).Decode(&result)
	pulls := result["pulls"].([]interface{})
	if len(pulls) != 0 {
		t.Fatalf("expected 0 pulls, got %d", len(pulls))
	}
}
