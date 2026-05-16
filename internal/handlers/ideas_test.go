package handlers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stpotter16/dump/internal/auth"
	"github.com/stpotter16/dump/internal/handlers"
	"github.com/stpotter16/dump/internal/handlers/authentication"
	"github.com/stpotter16/dump/internal/handlers/sessions"
	"github.com/stpotter16/dump/internal/store"
	"github.com/stpotter16/dump/internal/store/db"
	"github.com/stpotter16/dump/internal/store/sqlite"
)

type fakeEmbedder struct {
	embedding []float32
	err       error
}

func (f fakeEmbedder) Embed(_ context.Context, _ string) ([]float32, error) {
	return f.embedding, f.err
}

func newTestServer(t *testing.T, embedder handlers.Embedder) (*httptest.Server, store.Store) {
	t.Helper()

	d, err := db.New(t.TempDir())
	if err != nil {
		t.Fatalf("db.New: %v", err)
	}
	s, err := sqlite.New(d)
	if err != nil {
		t.Fatalf("sqlite.New: %v", err)
	}
	sm, err := sessions.New(d, func(key string) string {
		if key == sessions.SESSION_ENV_KEY {
			return "test-hmac-secret"
		}
		return ""
	})
	if err != nil {
		t.Fatalf("sessions.New: %v", err)
	}

	server := httptest.NewServer(handlers.NewServer(s, embedder, sm, authentication.New(s)))
	t.Cleanup(server.Close)
	return server, s
}

func loginUser(t *testing.T, server *httptest.Server, s store.Store, username, password string) []*http.Cookie {
	t.Helper()

	hash, err := auth.HashPassword(password)
	if err != nil {
		t.Fatalf("auth.HashPassword: %v", err)
	}
	if err := s.CreateUser(context.Background(), username, hash, false); err != nil {
		t.Fatalf("s.CreateUser: %v", err)
	}

	body, _ := json.Marshal(map[string]string{"username": username, "password": password})
	resp, err := http.Post(server.URL+"/login", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("POST /login: %v", err)
	}
	defer resp.Body.Close()
	if got, want := resp.StatusCode, http.StatusOK; got != want {
		t.Fatalf("POST /login status = %d, want %d", got, want)
	}

	return resp.Cookies()
}

var noRedirectClient = &http.Client{
	CheckRedirect: func(*http.Request, []*http.Request) error {
		return http.ErrUseLastResponse
	},
}

func withCookies(req *http.Request, cookies []*http.Cookie) *http.Request {
	for _, c := range cookies {
		req.AddCookie(c)
	}
	return req
}

// getCsrfToken loads the index page and extracts the CSRF token from the meta tag,
// mirroring how a browser would retrieve it before sending a mutating request.
func getCsrfToken(t *testing.T, server *httptest.Server, cookies []*http.Cookie) string {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, server.URL+"/", nil)
	resp, err := noRedirectClient.Do(withCookies(req, cookies))
	if err != nil {
		t.Fatalf("GET / for csrf token: %v", err)
	}
	defer resp.Body.Close()
	var buf bytes.Buffer
	buf.ReadFrom(resp.Body)
	body := buf.String()

	const marker = `name="csrf-token" content="`
	idx := strings.Index(body, marker)
	if idx == -1 {
		t.Fatalf("csrf-token meta tag not found in response")
	}
	start := idx + len(marker)
	end := strings.Index(body[start:], `"`)
	if end == -1 {
		t.Fatalf("could not parse csrf token from meta tag")
	}
	return body[start : start+end]
}

func postIdea(t *testing.T, server *httptest.Server, cookies []*http.Cookie, csrfToken, text string) *http.Response {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"text": text})
	req, _ := http.NewRequest(http.MethodPost, server.URL+"/ideas", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-CSRF-Token", csrfToken)
	resp, err := noRedirectClient.Do(withCookies(req, cookies))
	if err != nil {
		t.Fatalf("POST /ideas: %v", err)
	}
	return resp
}

func TestPostIdeas_Returns201OnSuccess(t *testing.T) {
	server, s := newTestServer(t, fakeEmbedder{embedding: []float32{1, 0, 0}})
	cookies := loginUser(t, server, s, "testuser", "password")
	csrfToken := getCsrfToken(t, server, cookies)

	resp := postIdea(t, server, cookies, csrfToken, "build a neural interface")
	defer resp.Body.Close()

	if got, want := resp.StatusCode, http.StatusCreated; got != want {
		t.Errorf("status = %d, want %d", got, want)
	}
}

func TestPostIdeas_ForbiddenWithoutCsrfToken(t *testing.T) {
	server, s := newTestServer(t, fakeEmbedder{})
	cookies := loginUser(t, server, s, "testuser", "password")

	body, _ := json.Marshal(map[string]string{"text": "sneaky request"})
	req, _ := http.NewRequest(http.MethodPost, server.URL+"/ideas", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := noRedirectClient.Do(withCookies(req, cookies))
	if err != nil {
		t.Fatalf("POST /ideas: %v", err)
	}
	defer resp.Body.Close()

	if got, want := resp.StatusCode, http.StatusForbidden; got != want {
		t.Errorf("status = %d, want %d", got, want)
	}
}

func TestPostIdeas_SavesIdeaEvenWhenEmbedderFails(t *testing.T) {
	server, s := newTestServer(t, fakeEmbedder{err: context.DeadlineExceeded})
	cookies := loginUser(t, server, s, "testuser", "password")
	csrfToken := getCsrfToken(t, server, cookies)

	resp := postIdea(t, server, cookies, csrfToken, "idea with no embedding")
	defer resp.Body.Close()

	if got, want := resp.StatusCode, http.StatusCreated; got != want {
		t.Fatalf("status = %d, want %d (idea should still be saved)", got, want)
	}

	ideas, err := s.GetIdeas(context.Background())
	if err != nil {
		t.Fatalf("GetIdeas: %v", err)
	}
	if got, want := len(ideas), 1; got != want {
		t.Errorf("len(ideas) = %d, want %d", got, want)
	}
	if ideas[0].Embedding != nil {
		t.Errorf("embedding should be nil when embedder fails")
	}
}

func TestReviewPage_ShowsSimilarIdeas(t *testing.T) {
	similar := []float32{0.99, 0.01, 0.0}
	unrelated := []float32{0.0, 0.0, 1.0}

	server, s := newTestServer(t, fakeEmbedder{})
	cookies := loginUser(t, server, s, "testuser", "password")

	ctx := context.Background()
	for _, idea := range []struct {
		text      string
		embedding []float32
	}{
		{"neural interface", similar},
		{"brain computer link", similar},
		{"morning run plan", unrelated},
	} {
		if _, err := s.CreateIdea(ctx, idea.text, idea.embedding); err != nil {
			t.Fatalf("CreateIdea %q: %v", idea.text, err)
		}
	}

	req, _ := http.NewRequest(http.MethodGet, server.URL+"/review", nil)
	resp, err := noRedirectClient.Do(withCookies(req, cookies))
	if err != nil {
		t.Fatalf("GET /review: %v", err)
	}
	defer resp.Body.Close()

	if got, want := resp.StatusCode, http.StatusOK; got != want {
		t.Fatalf("status = %d, want %d", got, want)
	}

	var buf bytes.Buffer
	buf.ReadFrom(resp.Body)
	body := buf.String()

	if !strings.Contains(body, "similar") {
		t.Errorf("expected review page to contain similarity links for similar embeddings")
	}
	if !strings.Contains(body, "morning run plan") {
		t.Errorf("expected review page to contain 'morning run plan'")
	}
}
