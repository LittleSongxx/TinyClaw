package knowledge

import (
	"strings"
	"testing"

	"github.com/LittleSongxx/TinyClaw/conf"
)

func TestFilterRetrievalHitsDropsWeakDenseOnlyMatches(t *testing.T) {
	hits := []RetrievalHit{
		{ChunkID: 1, DenseScore: 0.41, LexicalScore: 0, RRFScore: 0.016, FinalScore: 0.016},
		{ChunkID: 2, DenseScore: 0.39, LexicalScore: 0.01, RRFScore: 0.015, FinalScore: 0.015},
	}

	got := filterRetrievalHits(hits, retrievalThresholds{
		DenseMin:    0.55,
		LexicalMin:  0.05,
		FusedMin:    0.02,
		RerankerMin: 0.15,
	})
	if len(got) != 0 {
		t.Fatalf("expected weak hits to be filtered, got %+v", got)
	}
}

func TestFilterRetrievalHitsKeepsStrongEvidence(t *testing.T) {
	hits := []RetrievalHit{
		{ChunkID: 1, DenseScore: 0.72, LexicalScore: 0.01, RRFScore: 0.016, FinalScore: 0.016},
		{ChunkID: 2, DenseScore: 0.32, LexicalScore: 0.11, RRFScore: 0.018, FinalScore: 0.018},
	}

	got := filterRetrievalHits(hits, retrievalThresholds{
		DenseMin:    0.55,
		LexicalMin:  0.05,
		FusedMin:    0.02,
		RerankerMin: 0.15,
	})
	if len(got) != 2 {
		t.Fatalf("expected strong hits to remain, got %+v", got)
	}
}

func TestFilterRetrievalHitsUsesRerankerScore(t *testing.T) {
	hits := []RetrievalHit{
		{ChunkID: 1, DenseScore: 0.81, Reranked: true, RerankScore: 0.04, FinalScore: 0.04},
		{ChunkID: 2, DenseScore: 0.61, Reranked: true, RerankScore: 0.83, FinalScore: 0.83},
	}

	got := filterRetrievalHits(hits, retrievalThresholds{
		DenseMin:    0.55,
		LexicalMin:  0.05,
		FusedMin:    0.02,
		RerankerMin: 0.15,
	})
	if len(got) != 1 || got[0].ChunkID != 2 {
		t.Fatalf("expected reranker threshold to keep only chunk 2, got %+v", got)
	}
}

func TestNoRelevantKnowledgeAnswerFollowsLanguage(t *testing.T) {
	oldLang := conf.BaseConfInfo.Lang
	defer func() {
		conf.BaseConfInfo.Lang = oldLang
	}()

	conf.BaseConfInfo.Lang = "zh"
	if got := noRelevantKnowledgeAnswer(); !strings.Contains(got, "知识库里没有足够相关的内容") {
		t.Fatalf("expected zh fallback, got %q", got)
	}

	conf.BaseConfInfo.Lang = "en"
	if got := noRelevantKnowledgeAnswer(); !strings.Contains(got, "knowledge base does not contain enough relevant information") {
		t.Fatalf("expected en fallback, got %q", got)
	}
}
