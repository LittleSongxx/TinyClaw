package knowledge

import (
	"context"
	"errors"
	"time"

	"github.com/LittleSongxx/TinyClaw/conf"
	"github.com/LittleSongxx/TinyClaw/llm"
	"github.com/LittleSongxx/TinyClaw/logger"
	"github.com/LittleSongxx/langchaingo/embeddings"
	"github.com/LittleSongxx/langchaingo/llms"
	"github.com/LittleSongxx/langchaingo/llms/ernie"
	"github.com/LittleSongxx/langchaingo/llms/googleai"
	"github.com/LittleSongxx/langchaingo/llms/openai"
)

type Runtime struct {
	LLM *llm.LLM
}

func NewRuntime(options ...llm.Option) *Runtime {
	dp := &Runtime{
		LLM: llm.NewLLM(options...),
	}

	for _, o := range options {
		o(dp.LLM)
	}
	return dp
}

func (l *Runtime) Call(ctx context.Context, prompt string, options ...llms.CallOption) (string, error) {
	return llms.GenerateFromSinglePrompt(ctx, l, prompt, options...)
}

func (l *Runtime) GenerateContent(ctx context.Context, messages []llms.MessageContent, options ...llms.CallOption) (*llms.ContentResponse, error) {
	if Enabled() {
		prompt := ""
		for _, msg := range messages {
			for _, part := range msg.Parts {
				if txtPart, ok := part.(llms.TextContent); ok {
					prompt += txtPart.Text
				}
			}
		}
		return answerWithKnowledge(ctx, l.LLM, prompt, options...)
	}

	if err := l.LLM.CallLLM(); err != nil {
		logger.Error("error calling llm api", "err", err)
		return nil, errors.New("error calling llm api")
	}

	return &llms.ContentResponse{
		Choices: []*llms.ContentChoice{
			{Content: l.LLM.WholeContent},
		},
	}, nil
}

func Init() {
	if conf.KnowledgeConfInfo.EmbeddingType == "" {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	var err error
	switch conf.KnowledgeConfInfo.EmbeddingType {
	case "openai":
		conf.KnowledgeConfInfo.Embedder, err = initOpenAIEmbedding()
	case "gemini":
		conf.KnowledgeConfInfo.Embedder, err = initGeminiEmbedding(ctx)
	case "ernie":
		conf.KnowledgeConfInfo.Embedder, err = initErnieEmbedding()
	case "huggingface":
		conf.KnowledgeConfInfo.Embedder, err = initHuggingFaceEmbedding()
	default:
		logger.Error("embedding type not exist", "embedding_type", conf.KnowledgeConfInfo.EmbeddingType)
		return
	}

	if err != nil {
		logger.Error("init embedding fail", "err", err)
		return
	}

	if !conf.KnowledgeConfInfo.Enabled() {
		logger.Warn("knowledge pipeline is disabled",
			"feature_store", conf.KnowledgeConfInfo.FeatureStoreEnabled(),
			"object_storage", conf.KnowledgeConfInfo.ObjectStorageEnabled(),
			"queue", conf.KnowledgeConfInfo.QueueEnabled(),
		)
		return
	}

	if err = initKnowledge(ctx); err != nil {
		logger.Error("init knowledge service fail", "err", err)
	}
}

func initOpenAIEmbedding() (embeddings.Embedder, error) {
	llmEmbedder, err := openai.New(
		openai.WithToken(conf.BaseConfInfo.OpenAIToken),
	)
	if err != nil {
		return nil, err
	}
	return embeddings.NewEmbedder(llmEmbedder)
}

func initErnieEmbedding() (embeddings.Embedder, error) {
	llmEmbedder, err := ernie.New(
		ernie.WithModelName(ernie.ModelNameERNIEBot),
		ernie.WithAKSK(conf.BaseConfInfo.ErnieAK, conf.BaseConfInfo.ErnieSK),
	)
	if err != nil {
		return nil, err
	}
	return embeddings.NewEmbedder(llmEmbedder)
}

func initGeminiEmbedding(ctx context.Context) (embeddings.Embedder, error) {
	llmEmbedder, err := googleai.New(ctx,
		googleai.WithAPIKey(conf.BaseConfInfo.GeminiToken),
	)
	if err != nil {
		return nil, err
	}
	return embeddings.NewEmbedder(llmEmbedder)
}

func initHuggingFaceEmbedding() (embeddings.Embedder, error) {
	return newTEIEmbedder()
}
