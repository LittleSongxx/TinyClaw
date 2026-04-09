package knowledge

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestTEIRerankerUsesOfficialRerankEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/rerank" {
			t.Fatalf("expected /rerank path, got %s", r.URL.Path)
		}
		var payload map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if payload["query"] != "what is a node" {
			t.Fatalf("unexpected query payload: %+v", payload)
		}
		_ = json.NewEncoder(w).Encode([]map[string]interface{}{
			{"index": 1, "score": 0.91},
			{"index": 0, "score": 0.17},
		})
	}))
	defer server.Close()

	reranker := newTEIReranker(server.URL, 8)
	hits, err := reranker.Rerank(context.Background(), "what is a node", []RetrievalHit{
		{ChunkID: 10, Content: "foo", FinalScore: 0.03, RRFScore: 0.03},
		{ChunkID: 20, Content: "bar", FinalScore: 0.02, RRFScore: 0.02},
	})
	if err != nil {
		t.Fatalf("rerank failed: %v", err)
	}
	if len(hits) != 2 || hits[0].ChunkID != 20 || hits[0].RerankScore != 0.91 || !hits[0].Reranked {
		t.Fatalf("unexpected reranked hits: %+v", hits)
	}
}

func TestTEIRerankerSupportsResultsEnvelope(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]interface{}{
			"results": []map[string]interface{}{
				{"index": 0, "score": 0.77},
			},
		})
	}))
	defer server.Close()

	reranker := newTEIReranker(server.URL, 8)
	hits, err := reranker.Rerank(context.Background(), "question", []RetrievalHit{
		{ChunkID: 1, Content: "alpha", FinalScore: 0.01, RRFScore: 0.01},
	})
	if err != nil {
		t.Fatalf("rerank failed: %v", err)
	}
	if len(hits) != 1 || hits[0].RerankScore != 0.77 || !hits[0].Reranked {
		t.Fatalf("unexpected reranked hits: %+v", hits)
	}
}
