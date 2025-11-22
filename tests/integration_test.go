package api_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"pull-request-api.com/internal/api"
	"pull-request-api.com/internal/database"
	"pull-request-api.com/internal/models"
	"pull-request-api.com/internal/service"
)

const testConnString = "host=localhost port=55432 user=postgres password=postgres dbname=prdb_test sslmode=disable"

var testDB *sql.DB

func TestMain(m *testing.M) {
	var err error
	testDB, err = sql.Open("postgres", testConnString)
	if err != nil {
		log.Fatalf("Could not connect to test DB: %v", err)
	}

	if err := testDB.Ping(); err != nil {
		log.Printf("Skipping integration tests: DB not available (%v)", err)
		os.Exit(0)
	}

	teardownDB()
	if err := database.Migrate(testDB, "prdb_test", "file://../migrations"); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	code := m.Run()

	teardownDB()
	testDB.Close()
	os.Exit(code)
}

func teardownDB() {
	_, _ = testDB.Exec("TRUNCATE TABLE pr_reviewers, pull_requests, users, teams CASCADE")
}

func setupServer() (*chi.Mux, *service.Service) {
	svc := service.NewService(testDB)
	srv := api.NewServer(svc)
	r := chi.NewRouter()
	api.HandlerFromMux(srv, r)
	return r, svc
}

// --- ТЕСТЫ ---

func TestIntegration_FullFlow_CreateAndMerge(t *testing.T) {
	teardownDB()
	router, _ := setupServer()

	team := models.Team{
		TeamName: "Backend",
		Members: []models.TeamMember{
			{UserId: "alice", Username: "Alice", IsActive: true},
			{UserId: "bob", Username: "Bob", IsActive: true},
			{UserId: "charlie", Username: "Charlie", IsActive: true},
		},
	}
	postRequest(t, router, "/team/add", team, http.StatusOK)

	createPRBody := models.PostPullRequestCreateJSONRequestBody{
		PullRequestId:   "PR-100",
		PullRequestName: "Fix login bug",
		AuthorId:        "alice",
	}
	respPR := postRequest(t, router, "/pullRequest/create", createPRBody, http.StatusOK)

	var pr models.PullRequest
	json.Unmarshal(respPR, &pr)

	assert.Equal(t, "PR-100", pr.PullRequestId)
	assert.Equal(t, models.PullRequestStatusOPEN, pr.Status)
	assert.Contains(t, pr.AssignedReviewers, "bob")
	assert.Contains(t, pr.AssignedReviewers, "charlie")
	assert.Len(t, pr.AssignedReviewers, 2)

	mergeBody := models.PostPullRequestMergeJSONRequestBody{
		PullRequestId: "PR-100",
	}
	respMerge := postRequest(t, router, "/pullRequest/merge", mergeBody, http.StatusOK)

	json.Unmarshal(respMerge, &pr)
	assert.Equal(t, models.PullRequestStatusMERGED, pr.Status)
	assert.NotNil(t, pr.MergedAt)

	postRequest(t, router, "/pullRequest/merge", mergeBody, http.StatusOK)
}

func TestIntegration_ReassignReviewer(t *testing.T) {
	teardownDB()
	router, _ := setupServer()

	team := models.Team{
		TeamName: "Frontend",
		Members: []models.TeamMember{
			{UserId: "alice", Username: "Alice", IsActive: true},
			{UserId: "rev1", Username: "Reviewer 1", IsActive: true},
			{UserId: "rev2", Username: "Reviewer 2", IsActive: true},
			{UserId: "free_cand", Username: "Free Candidate", IsActive: true},
		},
	}
	postRequest(t, router, "/team/add", team, http.StatusOK)

	createPRBody := models.PostPullRequestCreateJSONRequestBody{
		PullRequestId: "PR-200", PullRequestName: "UI Update", AuthorId: "alice",
	}
	respPR := postRequest(t, router, "/pullRequest/create", createPRBody, http.StatusOK)

	var pr models.PullRequest
	json.Unmarshal(respPR, &pr)

	oldReviewer := pr.AssignedReviewers[0]

	reassignBody := models.PostPullRequestReassignJSONRequestBody{
		PullRequestId: "PR-200",
		OldUserId:     oldReviewer,
	}
	respReassign := postRequest(t, router, "/pullRequest/reassign", reassignBody, http.StatusOK)

	var prUpdated models.PullRequest
	json.Unmarshal(respReassign, &prUpdated)
	time.Sleep(1 * time.Second)
	assert.NotContains(t, prUpdated.AssignedReviewers, oldReviewer, "Старый ревьювер должен исчезнуть")
	assert.Contains(t, prUpdated.AssignedReviewers, "free_cand", "Свободный кандидат должен появиться (т.к. он единственный оставшийся)")
}

func TestIntegration_ErrorsAndEdgeCases(t *testing.T) {
	teardownDB()
	router, _ := setupServer()

	badPR := models.PostPullRequestCreateJSONRequestBody{
		PullRequestId: "PR-999", PullRequestName: "Ghost PR", AuthorId: "ghost",
	}
	respErr := postRequest(t, router, "/pullRequest/create", badPR, http.StatusNotFound)
	assert.Contains(t, string(respErr), string(models.NOTFOUND))

	setupUser(t, router)
	goodPR := models.PostPullRequestCreateJSONRequestBody{
		PullRequestId: "PR-1", PullRequestName: "First", AuthorId: "u1",
	}
	postRequest(t, router, "/pullRequest/create", goodPR, http.StatusOK)

	respConflict := postRequest(t, router, "/pullRequest/create", goodPR, http.StatusConflict)
	assert.Contains(t, string(respConflict), string(models.PREXISTS))

	postRequest(t, router, "/pullRequest/merge", models.PostPullRequestMergeJSONRequestBody{PullRequestId: "PR-1"}, http.StatusOK)

	reassign := models.PostPullRequestReassignJSONRequestBody{PullRequestId: "PR-1", OldUserId: "u2"} // u2 был создан в setupUser
	respMerged := postRequest(t, router, "/pullRequest/reassign", reassign, http.StatusBadRequest)
	assert.Contains(t, string(respMerged), string(models.PRMERGED))
}

// --- Хэлперы ---

func postRequest(t *testing.T, router *chi.Mux, path string, body any, expectedStatus int) []byte {
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}

	req := httptest.NewRequest(http.MethodPost, path, &buf)
	req.Header.Set("Content-Type", "application/json")

	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	require.Equal(t, expectedStatus, rec.Code, "Path: %s, Response: %s", path, rec.Body.String())
	return rec.Body.Bytes()
}

func setupUser(t *testing.T, router *chi.Mux) {
	team := models.Team{
		TeamName: "T1",
		Members: []models.TeamMember{
			{UserId: "u1", Username: "User1", IsActive: true},
			{UserId: "u2", Username: "User2", IsActive: true},
			{UserId: "u3", Username: "User3", IsActive: true},
		},
	}
	postRequest(t, router, "/team/add", team, http.StatusOK)
}
