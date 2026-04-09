package knowledge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/LittleSongxx/TinyClaw/conf"
	"github.com/LittleSongxx/langchaingo/embeddings"
)

const teiBatchSize = 32

type teiClient struct {
	baseURL    string
	modelID    string
	httpClient *http.Client
}

type teiEmbedder struct {
	client           *teiClient
	queryInstruction string
}

type teiEmbedRequest struct {
	Inputs any `json:"inputs"`
}

func newTEIEmbedder() (embeddings.Embedder, error) {
	baseURL := strings.TrimSpace(conf.KnowledgeConfInfo.EmbeddingBaseURL)
	if baseURL == "" {
		return nil, fmt.Errorf("embedding_base_url is required when embedding_type=huggingface")
	}

	return &teiEmbedder{
		client: &teiClient{
			baseURL: strings.TrimRight(baseURL, "/"),
			modelID: strings.TrimSpace(conf.KnowledgeConfInfo.EmbeddingModelID),
			httpClient: &http.Client{
				Transport: http.DefaultTransport,
			},
		},
		queryInstruction: strings.TrimSpace(conf.KnowledgeConfInfo.EmbeddingQueryInstruction),
	}, nil
}

func (e *teiEmbedder) EmbedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	texts = embeddings.MaybeRemoveNewLines(texts, true)
	return embeddings.BatchedEmbed(ctx, embeddings.EmbedderClientFunc(e.client.embedDocuments), texts, teiBatchSize)
}

func (e *teiEmbedder) EmbedQuery(ctx context.Context, text string) ([]float32, error) {
	text = strings.ReplaceAll(text, "\n", " ")
	if e.queryInstruction != "" && !strings.HasPrefix(text, e.queryInstruction) {
		text = e.queryInstruction + text
	}
	return e.client.embedQuery(ctx, text)
}

func (c *teiClient) embedDocuments(ctx context.Context, texts []string) ([][]float32, error) {
	raw, err := c.doEmbed(ctx, texts)
	if err != nil {
		return nil, err
	}

	var vectors [][]float32
	if err := json.Unmarshal(raw, &vectors); err == nil {
		return vectors, nil
	}

	var vector []float32
	if err := json.Unmarshal(raw, &vector); err == nil && len(texts) == 1 {
		return [][]float32{vector}, nil
	}

	return nil, fmt.Errorf("unexpected TEI embeddings response for model %q", c.modelID)
}

func (c *teiClient) embedQuery(ctx context.Context, text string) ([]float32, error) {
	raw, err := c.doEmbed(ctx, text)
	if err != nil {
		return nil, err
	}

	var vector []float32
	if err := json.Unmarshal(raw, &vector); err == nil {
		return vector, nil
	}

	var vectors [][]float32
	if err := json.Unmarshal(raw, &vectors); err == nil && len(vectors) > 0 {
		return vectors[0], nil
	}

	return nil, fmt.Errorf("unexpected TEI query response for model %q", c.modelID)
}

func (c *teiClient) doEmbed(ctx context.Context, inputs any) (json.RawMessage, error) {
	body, err := json.Marshal(teiEmbedRequest{Inputs: inputs})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.embedURL(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("tei embed request failed: status=%s body=%s", resp.Status, strings.TrimSpace(string(respBody)))
	}

	return json.RawMessage(respBody), nil
}

func (c *teiClient) embedURL() string {
	if strings.HasSuffix(c.baseURL, "/embed") {
		return c.baseURL
	}
	return c.baseURL + "/embed"
}
