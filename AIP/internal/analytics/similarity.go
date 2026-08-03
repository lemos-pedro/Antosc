package analytics

import (
	"math"
	"sort"

	"github.com/antosc/aip/internal/repository/postgres"
)

type SimilarTower struct {
	TowerID    string  `json:"tower_id"`
	Similarity float64 `json:"similarity"`
}

func Cosine(a, b []float64) float64 {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	if n == 0 {
		return 0
	}
	var dot, na, nb float64
	for i := 0; i < n; i++ {
		dot += a[i] * b[i]
		na += a[i] * a[i]
		nb += b[i] * b[i]
	}
	if na == 0 || nb == 0 {
		return 0
	}
	return dot / (math.Sqrt(na) * math.Sqrt(nb))
}

func TopSimilar(target postgres.TowerEmbedding, all []postgres.TowerEmbedding, k int) []SimilarTower {
	if k <= 0 {
		k = 5
	}
	type pair struct {
		id string
		s  float64
	}
	var ranked []pair
	for _, e := range all {
		if e.TowerID == target.TowerID {
			continue
		}
		ranked = append(ranked, pair{e.TowerID, Cosine(target.Vector, e.Vector)})
	}
	sort.Slice(ranked, func(i, j int) bool { return ranked[i].s > ranked[j].s })
	if len(ranked) > k {
		ranked = ranked[:k]
	}
	out := make([]SimilarTower, 0, len(ranked))
	for _, p := range ranked {
		out = append(out, SimilarTower{TowerID: p.id, Similarity: math.Round(p.s*1000) / 1000})
	}
	return out
}
