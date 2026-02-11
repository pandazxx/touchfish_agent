package github

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func setupServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *RESTClient) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	client := NewRESTClientWithBaseURL(srv.URL, "myorg", "myrepo", "test-token")
	return srv, client
}

func jsonResponse(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(body)
}

func TestCreatePR(t *testing.T) {
	_, client := setupServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !strings.HasSuffix(r.URL.Path, "/pulls") {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("missing auth header")
		}
		jsonResponse(w, 201, ghPullRequest{
			Number: 1,
			Title:  "Test PR",
			Body:   "body",
			State:  "open",
			Head:   struct{ Ref string `json:"ref"` }{Ref: "feature"},
			Base:   struct{ Ref string `json:"ref"` }{Ref: "master"},
		})
	})

	pr, err := client.CreatePR(context.Background(), "feature", "master", "Test PR", "body")
	if err != nil {
		t.Fatalf("CreatePR: %v", err)
	}
	if pr.Number != 1 {
		t.Errorf("expected number=1, got %d", pr.Number)
	}
	if pr.Head != "feature" {
		t.Errorf("expected head=feature, got %q", pr.Head)
	}
}

func TestGetPR(t *testing.T) {
	_, client := setupServer(t, func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, "/pulls/42") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		jsonResponse(w, 200, ghPullRequest{
			Number: 42,
			Title:  "My PR",
			State:  "open",
		})
	})

	pr, err := client.GetPR(context.Background(), 42)
	if err != nil {
		t.Fatalf("GetPR: %v", err)
	}
	if pr.Number != 42 {
		t.Errorf("expected number=42, got %d", pr.Number)
	}
}

func TestListPRs(t *testing.T) {
	_, client := setupServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("state") != "open" {
			t.Errorf("expected state=open, got %q", r.URL.Query().Get("state"))
		}
		jsonResponse(w, 200, []ghPullRequest{
			{Number: 1, Title: "PR 1", State: "open"},
			{Number: 2, Title: "PR 2", State: "open"},
		})
	})

	prs, err := client.ListPRs(context.Background(), PRStateOpen, "")
	if err != nil {
		t.Fatalf("ListPRs: %v", err)
	}
	if len(prs) != 2 {
		t.Fatalf("expected 2 PRs, got %d", len(prs))
	}
}

func TestListIssues(t *testing.T) {
	_, client := setupServer(t, func(w http.ResponseWriter, r *http.Request) {
		labels := r.URL.Query().Get("labels")
		if labels != "Agent to fix" {
			t.Errorf("expected labels='Agent to fix', got %q", labels)
		}
		jsonResponse(w, 200, []ghIssue{
			{
				Number: 10,
				Title:  "Bug",
				State:  "open",
				Labels: []struct{ Name string `json:"name"` }{{Name: "Agent to fix"}},
			},
		})
	})

	issues, err := client.ListIssues(context.Background(), []string{"Agent to fix"})
	if err != nil {
		t.Fatalf("ListIssues: %v", err)
	}
	if len(issues) != 1 {
		t.Fatalf("expected 1 issue, got %d", len(issues))
	}
	if issues[0].Labels[0] != "Agent to fix" {
		t.Errorf("expected label 'Agent to fix', got %q", issues[0].Labels[0])
	}
}

func TestGetIssue(t *testing.T) {
	_, client := setupServer(t, func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 200, ghIssue{
			Number: 5,
			Title:  "Issue 5",
			Body:   "details",
			State:  "open",
		})
	})

	issue, err := client.GetIssue(context.Background(), 5)
	if err != nil {
		t.Fatalf("GetIssue: %v", err)
	}
	if issue.Number != 5 || issue.Body != "details" {
		t.Errorf("unexpected issue: %+v", issue)
	}
}

func TestCreateIssue(t *testing.T) {
	_, client := setupServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		jsonResponse(w, 201, ghIssue{
			Number: 20,
			Title:  "New Issue",
			State:  "open",
			Labels: []struct{ Name string `json:"name"` }{{Name: "bug"}},
		})
	})

	issue, err := client.CreateIssue(context.Background(), "New Issue", "body", []string{"bug"})
	if err != nil {
		t.Fatalf("CreateIssue: %v", err)
	}
	if issue.Number != 20 {
		t.Errorf("expected number=20, got %d", issue.Number)
	}
}

func TestAddIssueLabel(t *testing.T) {
	_, client := setupServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || !strings.Contains(r.URL.Path, "/labels") {
			t.Errorf("unexpected: %s %s", r.Method, r.URL.Path)
		}
		jsonResponse(w, 200, nil)
	})

	err := client.AddIssueLabel(context.Background(), 10, "Agent fixing")
	if err != nil {
		t.Fatalf("AddIssueLabel: %v", err)
	}
}

func TestRemoveIssueLabel(t *testing.T) {
	_, client := setupServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			t.Errorf("expected DELETE, got %s", r.Method)
		}
		if !strings.HasSuffix(r.URL.Path, "/labels/Agent%20to%20fix") &&
			!strings.HasSuffix(r.URL.Path, "/labels/Agent to fix") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.WriteHeader(200)
	})

	err := client.RemoveIssueLabel(context.Background(), 10, "Agent to fix")
	if err != nil {
		t.Fatalf("RemoveIssueLabel: %v", err)
	}
}

func TestListIssueComments(t *testing.T) {
	_, client := setupServer(t, func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 200, []ghComment{
			{
				ID:   1,
				Body: "first comment",
				User: struct{ Login string `json:"login"` }{Login: "user1"},
				CreatedAt: "2024-01-01T00:00:00Z",
			},
		})
	})

	comments, err := client.ListIssueComments(context.Background(), 5)
	if err != nil {
		t.Fatalf("ListIssueComments: %v", err)
	}
	if len(comments) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(comments))
	}
	if comments[0].Author != "user1" {
		t.Errorf("expected author=user1, got %q", comments[0].Author)
	}
}

func TestListCommits(t *testing.T) {
	_, client := setupServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("sha") != "main" {
			t.Errorf("expected sha=main, got %q", r.URL.Query().Get("sha"))
		}
		jsonResponse(w, 200, []ghCommit{
			{
				SHA: "abc123",
				Commit: struct {
					Message string `json:"message"`
					Author  struct {
						Name string `json:"name"`
						Date string `json:"date"`
					} `json:"author"`
				}{
					Message: "initial",
					Author: struct {
						Name string `json:"name"`
						Date string `json:"date"`
					}{Name: "Dev", Date: "2024-01-01T00:00:00Z"},
				},
			},
		})
	})

	commits, err := client.ListCommits(context.Background(), "main", 10)
	if err != nil {
		t.Fatalf("ListCommits: %v", err)
	}
	if len(commits) != 1 {
		t.Fatalf("expected 1 commit, got %d", len(commits))
	}
	if commits[0].SHA != "abc123" {
		t.Errorf("expected sha=abc123, got %q", commits[0].SHA)
	}
}

func TestAPIError(t *testing.T) {
	_, client := setupServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		w.Write([]byte(`{"message": "Not Found"}`))
	})

	_, err := client.GetPR(context.Background(), 999)
	if err == nil {
		t.Fatal("expected error for 404")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("expected 404 in error, got: %v", err)
	}
}

func TestUpdatePR(t *testing.T) {
	_, client := setupServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Errorf("expected PATCH, got %s", r.Method)
		}
		w.WriteHeader(200)
	})

	title := "Updated"
	err := client.UpdatePR(context.Background(), 1, &title, nil)
	if err != nil {
		t.Fatalf("UpdatePR: %v", err)
	}
}

func TestAddPRComment(t *testing.T) {
	_, client := setupServer(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		jsonResponse(w, 201, nil)
	})

	err := client.AddPRComment(context.Background(), 1, "LGTM")
	if err != nil {
		t.Fatalf("AddPRComment: %v", err)
	}
}

func TestAddIssueComment(t *testing.T) {
	_, client := setupServer(t, func(w http.ResponseWriter, r *http.Request) {
		jsonResponse(w, 201, nil)
	})

	err := client.AddIssueComment(context.Background(), 5, "Working on it")
	if err != nil {
		t.Fatalf("AddIssueComment: %v", err)
	}
}
