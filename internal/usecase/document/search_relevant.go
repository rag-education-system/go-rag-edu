package document

import (
	"context"
	"fmt"

	"rag-api/internal/domain/entity"
)

func (uc *DocumentUsecase) searchRelevantChunks(
	ctx context.Context,
	query string,
	history []ChatMessage,
) ([]entity.SimilarChunk, error) {
	searchCandidates := []string{query}
	if contextual := buildSearchQuery(query, history); contextual != query {
		searchCandidates = append(searchCandidates, contextual)
	}

	thresholds := []float64{uc.threshold, 0.4, 0.3}

	for _, candidate := range searchCandidates {
		embeddings, err := uc.embedder.GenerateBatchEmbeddings(ctx, []string{candidate})
		if err != nil {
			return nil, fmt.Errorf("failed to generate query embedding: %w", err)
		}
		if len(embeddings) == 0 {
			continue
		}

		for _, threshold := range thresholds {
			chunks, err := uc.chunkRepo.SearchSimilar(ctx, embeddings[0], uc.topK, threshold)
			if err != nil {
				return nil, fmt.Errorf("failed to search similar chunks: %w", err)
			}
			if len(chunks) > 0 {
				return chunks, nil
			}
		}
	}

	terms := extractSearchTerms(query)
	if len(terms) > 0 {
		chunks, err := uc.chunkRepo.SearchByKeywords(ctx, terms, uc.topK)
		if err != nil {
			return nil, fmt.Errorf("failed to search chunks by keywords: %w", err)
		}
		if len(chunks) > 0 {
			return chunks, nil
		}
	}

	return nil, nil
}
