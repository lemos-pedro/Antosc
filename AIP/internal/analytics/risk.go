package analytics

import (
	"math"
	"math/rand"
	"sort"
	"time"

	"github.com/antosc/aip/internal/repository/postgres"
)

// PortfolioRisk é o resultado Monte Carlo a nível de rede.
type PortfolioRisk struct {
	HorizonDays       int     `json:"horizon_days"`
	NSimulations      int     `json:"n_simulations"`
	Sites             int     `json:"sites"`
	ExpectedFailures  float64 `json:"expected_failures"`
	P50Failures       float64 `json:"p50_failures"`
	P90Failures       float64 `json:"p90_failures"`
	P99Failures       float64 `json:"p99_failures"`
	ProbAtLeastOne    float64 `json:"prob_at_least_one_failure"`
	MeanFailureProb   float64 `json:"mean_site_failure_prob"`
	GeneratedAt       string  `json:"generated_at"`
}

// SiteFailureProb mapeia previsão → P(falha no horizonte).
func SiteFailureProb(p postgres.Prediction, horizonDays int) float64 {
	if horizonDays <= 0 {
		horizonDays = 30
	}
	status := p.Status
	switch status {
	case "critical_now", "critical":
		return clamp01(0.85 + (100-p.Score)/500)
	case "at_risk", "anomaly", "warning":
		base := 0.35 + (100-p.Score)/300
		return clamp01(base)
	case "watch":
		return clamp01(0.15 + (100-p.Score)/400)
	case "healthy", "normal", "stable":
		return clamp01(0.03 + (100-p.Score)/800)
	}
	// fallback pelo score
	if p.Score <= 20 {
		return 0.7
	}
	if p.Score <= 50 {
		return 0.35
	}
	if p.Score <= 80 {
		return 0.12
	}
	return 0.04
}

func clamp01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

// MonteCarloPortfolio simula nº de falhas no horizonte.
func MonteCarloPortfolio(preds []postgres.Prediction, horizonDays, nSim int, seed int64) PortfolioRisk {
	if nSim <= 0 {
		nSim = 5000
	}
	if horizonDays <= 0 {
		horizonDays = 30
	}
	// uma previsão por torre (já deve vir ListLatestPerTower)
	probs := make([]float64, 0, len(preds))
	for _, p := range preds {
		probs = append(probs, SiteFailureProb(p, horizonDays))
	}
	rng := rand.New(rand.NewSource(seed))
	if seed == 0 {
		rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}

	counts := make([]float64, nSim)
	atLeast := 0
	var sumExpected float64
	for _, pr := range probs {
		sumExpected += pr
	}

	for i := 0; i < nSim; i++ {
		nFail := 0
		for _, pr := range probs {
			if rng.Float64() < pr {
				nFail++
			}
		}
		counts[i] = float64(nFail)
		if nFail > 0 {
			atLeast++
		}
	}
	sort.Float64s(counts)

	percentile := func(p float64) float64 {
		if len(counts) == 0 {
			return 0
		}
		idx := int(math.Ceil(p*float64(len(counts)))) - 1
		if idx < 0 {
			idx = 0
		}
		if idx >= len(counts) {
			idx = len(counts) - 1
		}
		return counts[idx]
	}

	meanProb := 0.0
	if len(probs) > 0 {
		meanProb = sumExpected / float64(len(probs))
	}

	return PortfolioRisk{
		HorizonDays:      horizonDays,
		NSimulations:     nSim,
		Sites:            len(probs),
		ExpectedFailures: math.Round(sumExpected*100) / 100,
		P50Failures:      percentile(0.50),
		P90Failures:      percentile(0.90),
		P99Failures:      percentile(0.99),
		ProbAtLeastOne:   math.Round(float64(atLeast)/float64(nSim)*1000) / 1000,
		MeanFailureProb:  math.Round(meanProb*1000) / 1000,
		GeneratedAt:      time.Now().UTC().Format(time.RFC3339),
	}
}
