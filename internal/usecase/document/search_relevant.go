package document

import (
	"context"
	"fmt"
	"strings"

	"rag-api/internal/domain/entity"
)

func (uc *DocumentUsecase) searchRelevantChunks(
	ctx context.Context,
	userID string,
	originalQuery string,
	searchQuery string,
	history []ChatMessage,
) ([]entity.SimilarChunk, string, error) {
	searchCandidates := uniqueStrings([]string{searchQuery, originalQuery})
	if contextual := buildSearchQuery(originalQuery, history); contextual != originalQuery && contextual != searchQuery {
		searchCandidates = append(searchCandidates, contextual)
	}

	seenChunks := make(map[string]entity.SimilarChunk)
	expandedTopK := uc.topK * 3
	searchType := "vector"

	// Hybrid search with primary query (AI-Hukum-BE pattern)
	if uc.useHybridSearch {
		embeddings, err := uc.embedder.GenerateBatchEmbeddings(ctx, []string{searchQuery})
		if err != nil {
			return nil, "", fmt.Errorf("failed to generate query embedding: %w", err)
		}
		if len(embeddings) > 0 {
			hybridChunks, err := uc.chunkRepo.HybridSearchWithAccess(ctx, searchQuery, embeddings[0], userID, expandedTopK, uc.threshold)
			if err == nil && len(hybridChunks) > 0 {
				searchType = "hybrid"
				for _, chunk := range hybridChunks {
					if existing, ok := seenChunks[chunk.ID]; !ok || chunk.Similarity > existing.Similarity {
						seenChunks[chunk.ID] = chunk
					}
				}
			}
		}
	}

	for _, candidate := range searchCandidates {
		embeddings, err := uc.embedder.GenerateBatchEmbeddings(ctx, []string{candidate})
		if err != nil {
			return nil, "", fmt.Errorf("failed to generate query embedding: %w", err)
		}
		if len(embeddings) == 0 {
			continue
		}

		chunks, err := uc.chunkRepo.SearchSimilarWithAccess(ctx, embeddings[0], userID, expandedTopK, uc.threshold)
		if err != nil {
			return nil, "", fmt.Errorf("failed to search similar chunks: %w", err)
		}

		for _, chunk := range chunks {
			if existing, ok := seenChunks[chunk.ID]; !ok || chunk.Similarity > existing.Similarity {
				seenChunks[chunk.ID] = chunk
			}
		}
	}

	terms := extractSearchTerms(originalQuery)
	if len(terms) > 0 {
		keywordChunks, err := uc.chunkRepo.SearchByKeywords(ctx, terms, userID, expandedTopK)
		if err != nil {
			return nil, "", fmt.Errorf("failed to search chunks by keywords: %w", err)
		}

		for _, chunk := range keywordChunks {
			if existing, ok := seenChunks[chunk.ID]; !ok || chunk.Similarity > existing.Similarity {
				seenChunks[chunk.ID] = chunk
			}
		}
	}

	if len(seenChunks) == 0 {
		return nil, searchType, nil
	}

	result := make([]entity.SimilarChunk, 0, len(seenChunks))
	for _, chunk := range seenChunks {
		result = append(result, chunk)
	}

	sortBySimilarity(result)

	if len(result) > uc.topK*2 {
		result = result[:uc.topK*2]
	}

	return result, searchType, nil
}

func uniqueStrings(items []string) []string {
	seen := make(map[string]bool)
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		out = append(out, item)
	}
	return out
}

func sortBySimilarity(chunks []entity.SimilarChunk) {
	for i := 0; i < len(chunks)-1; i++ {
		for j := i + 1; j < len(chunks); j++ {
			if chunks[j].Similarity > chunks[i].Similarity {
				chunks[i], chunks[j] = chunks[j], chunks[i]
			}
		}
	}
}
