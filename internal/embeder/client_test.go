package embeder_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stpotter16/dump/internal/embeder"
)

func newEmbedServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, embeder.Client) {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client := embeder.New(server.URL, "test-key", embeder.WithRetryDelays(0, 0, 0))
	return server, client
}

func TestEmbed_ReturnsEmbeddingOnSuccess(t *testing.T) {
	want := []float32{0.1, 0.2, 0.3}
	_, client := newEmbedServer(t, func(w http.ResponseWriter, r *http.Request) {
		if got, wantMethod := r.Method, http.MethodPost; got != wantMethod {
			t.Errorf("method = %q, want %q", got, wantMethod)
		}
		if got, wantPath := r.URL.Path, "/embed"; got != wantPath {
			t.Errorf("path = %q, want %q", got, wantPath)
		}
		if got, wantKey := r.Header.Get("X-API-Key"), "test-key"; got != wantKey {
			t.Errorf("X-API-Key = %q, want %q", got, wantKey)
		}
		json.NewEncoder(w).Encode(map[string]any{"embedding": want})
	})

	got, err := client.Embed(context.Background(), "hello")
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if gotLen, wantLen := len(got), len(want); gotLen != wantLen {
		t.Fatalf("len(embedding) = %d, want %d", gotLen, wantLen)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("embedding[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}

func TestEmbed_RetriesOn503ThenSucceeds(t *testing.T) {
	want := []float32{0.5, 0.5}
	attempts := 0

	_, client := newEmbedServer(t, func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"embedding": want})
	})

	got, err := client.Embed(context.Background(), "retry me")
	if err != nil {
		t.Fatalf("Embed: %v", err)
	}
	if got, want := attempts, 3; got != want {
		t.Errorf("attempts = %d, want %d", got, want)
	}
	if len(got) == 0 {
		t.Errorf("expected non-empty embedding")
	}
}

func TestEmbed_ErrorsAfterExhaustedRetries(t *testing.T) {
	attempts := 0
	_, client := newEmbedServer(t, func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusServiceUnavailable)
	})

	_, err := client.Embed(context.Background(), "always unavailable")
	if err == nil {
		t.Fatal("expected error when all retries exhausted, got nil")
	}
	if got, want := attempts, 4; got != want {
		t.Errorf("attempts = %d, want %d (initial + 3 retries)", got, want)
	}
}

func TestEmbed_DoesNotRetryNon503Errors(t *testing.T) {
	attempts := 0
	_, client := newEmbedServer(t, func(w http.ResponseWriter, r *http.Request) {
		attempts++
		w.WriteHeader(http.StatusUnauthorized)
	})

	_, err := client.Embed(context.Background(), "unauthorized")
	if err == nil {
		t.Fatal("expected error for 401 response, got nil")
	}
	if got, want := attempts, 1; got != want {
		t.Errorf("attempts = %d, want %d (401 must not retry)", got, want)
	}
}
