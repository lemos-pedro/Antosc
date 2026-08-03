package analytics

import (
	"testing"

	"github.com/antosc/aip/internal/repository/postgres"
)

func TestMonteCarloPortfolio(t *testing.T) {
	preds := []postgres.Prediction{
		{TowerID: "a", Score: 10, Status: "critical"},
		{TowerID: "b", Score: 90, Status: "healthy"},
		{TowerID: "c", Score: 40, Status: "at_risk"},
	}
	r := MonteCarloPortfolio(preds, 30, 2000, 42)
	if r.Sites != 3 {
		t.Fatalf("sites %d", r.Sites)
	}
	if r.ExpectedFailures <= 0 {
		t.Fatalf("expected > 0, got %v", r.ExpectedFailures)
	}
}
