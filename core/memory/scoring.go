package memory

import "sort"

// scoredCandidate 是 score_and_rank 的中间结果。
type scoredCandidate struct {
	id      string
	score   float64
	payload map[string]any
	details map[string]any
}

// scoreAndRank 对齐 utils/scoring.score_and_rank:
// 阈值先滤语义分, combined = (semantic + bm25 + entity) / max_possible, 封顶 1.0。
type rankCandidate struct {
	id      string
	score   float64
	payload map[string]any
}

func scoreAndRank(candidates []rankCandidate, bm25Scores, entityBoosts map[string]float64, threshold float64, topK int, explain bool) []scoredCandidate {
	maxPossible := 1.0
	if len(bm25Scores) > 0 {
		maxPossible += 1.0
	}
	if len(entityBoosts) > 0 {
		maxPossible += entityBoostWeight
	}
	var scored []scoredCandidate
	for _, c := range candidates {
		if c.score < threshold {
			continue
		}
		bm25 := bm25Scores[c.id]
		boost := entityBoosts[c.id]
		raw := c.score + bm25 + boost
		combined := raw / maxPossible
		if combined > 1.0 {
			combined = 1.0
		}
		sc := scoredCandidate{id: c.id, score: combined, payload: c.payload}
		if explain {
			sc.details = map[string]any{
				"semantic_score":     c.score,
				"bm25_score":         bm25,
				"entity_boost":       boost,
				"raw_score":          raw,
				"max_possible_score": maxPossible,
				"final_score":        combined,
				"threshold":          threshold,
			}
		}
		scored = append(scored, sc)
	}
	sort.SliceStable(scored, func(i, j int) bool { return scored[i].score > scored[j].score })
	if len(scored) > topK {
		scored = scored[:topK]
	}
	return scored
}
