package knowledge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
)

type teiReranker struct {
	baseURL    string
	httpClient *http.Client
	topN       int
}

type teiRerankRequest struct {
	Query     string   `json:"query"`
	Texts     []string `json:"texts"`
	RawScores bool     `json:"raw_scores"`
}

type teiRerankItem struct {
	Index int     `json:"index"`
	Score float64 `json:"score"`
	Text  string  `json:"text,omitempty"`
}

type teiRerankEnvelope struct {
	Results []teiRerankItem `json:"results"`
}

func newTEIReranker(baseURL string, topN int) Reranker {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return noopReranker{}
	}
	if topN <= 0 {
		topN = 8
	}
	return &teiReranker{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Transport: http.DefaultTransport,
		},
		topN: topN,
	}
}

func (r *teiReranker) Rerank(ctx context.Context, query string, hits []RetrievalHit) ([]RetrievalHit, error) {
	if r == nil || strings.TrimSpace(query) == "" || len(hits) == 0 {
		return hits, nil
	}

	if len(hits) > r.topN {
		hits = append([]RetrievalHit(nil), hits[:r.topN]...)
	} else {
		hits = append([]RetrievalHit(nil), hits...)
	}

	texts := make([]string, 0, len(hits))
	for _, hit := range hits {
		texts = append(texts, hit.Content)
	}

	scores, err := r.doRerank(ctx, query, texts)
	if err != nil {
		return nil, err
	}

	indexed := make([]RetrievalHit, 0, len(scores))
	for _, item := range scores {
		if item.Index < 0 || item.Index >= len(hits) {
			continue
		}
		hit := hits[item.Index]
		hit.RerankScore = item.Score
		hit.FinalScore = item.Score
		hit.Reranked = true
		indexed = append(indexed, hit)
	}
	if len(indexed) == 0 {
		return hits, nil
	}

	sort.Slice(indexed, func(i, j int) bool {
		if indexed[i].FinalScore == indexed[j].FinalScore {
			return indexed[i].RRFScore > indexed[j].RRFScore
		}
		return indexed[i].FinalScore > indexed[j].FinalScore
	})
	return indexed, nil
}

func (r *teiReranker) doRerank(ctx context.Context, query string, texts []string) ([]teiRerankItem, error) {
	body, err := json.Marshal(teiRerankRequest{
		Query:     query,
		Texts:     texts,
		RawScores: false,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, r.rerankURL(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("tei rerank request failed: status=%s body=%s", resp.Status, strings.TrimSpace(string(respBody)))
	}

	var items []teiRerankItem
	if err := json.Unmarshal(respBody, &items); err == nil {
		return items, nil
	}

	var envelope teiRerankEnvelope
	if err := json.Unmarshal(respBody, &envelope); err == nil && len(envelope.Results) > 0 {
		return envelope.Results, nil
	}

	return nil, fmt.Errorf("unexpected TEI rerank response: %s", strings.TrimSpace(string(respBody)))
}

func (r *teiReranker) rerankURL() string {
	if strings.HasSuffix(r.baseURL, "/rerank") {
		return r.baseURL
	}
	return r.baseURL + "/rerank"
}
