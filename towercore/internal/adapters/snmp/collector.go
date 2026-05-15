package snmp

import (
	"context"
	"errors"
	"strings"

	"towercore/internal/core/domain"
)

type Collector interface {
	Collect(ctx context.Context, tower domain.Tower, profile Profile) (map[string]float64, error)
}

type SyntheticCollector struct{}

func NewSyntheticCollector() *SyntheticCollector {
	return &SyntheticCollector{}
}

func (c *SyntheticCollector) Collect(_ context.Context, _ domain.Tower, profile Profile) (map[string]float64, error) {
	out := make(map[string]float64, len(profile.Metrics))
	for _, md := range profile.Metrics {
		out[md.OID] = baselineValue(md.Key, md.Scale)
	}
	return out, nil
}

type FallbackCollector struct {
	primary  Collector
	fallback Collector
}

func NewFallbackCollector(primary, fallback Collector) *FallbackCollector {
	return &FallbackCollector{primary: primary, fallback: fallback}
}

func (c *FallbackCollector) Collect(ctx context.Context, tower domain.Tower, profile Profile) (map[string]float64, error) {
	if c.primary == nil && c.fallback == nil {
		return nil, errors.New("collector not configured")
	}
	if c.primary != nil {
		if out, err := c.primary.Collect(ctx, tower, profile); err == nil {
			return out, nil
		}
	}
	if c.fallback != nil {
		return c.fallback.Collect(ctx, tower, profile)
	}
	return nil, errors.New("collection failed")
}

func baselineValue(key string, scale float64) float64 {
	k := strings.ToLower(key)
	switch {
	case strings.Contains(k, "temperature"):
		return 34 / safeScale(scale)
	case strings.Contains(k, "voltage"):
		return 4850 / safeScale(scale)
	case strings.Contains(k, "current"):
		return 125 / safeScale(scale)
	case strings.Contains(k, "remaining"):
		return 78 / safeScale(scale)
	case strings.Contains(k, "heartbeat"):
		return 60 / safeScale(scale)
	default:
		return 1 / safeScale(scale)
	}
}

func safeScale(scale float64) float64 {
	if scale == 0 {
		return 1
	}
	return scale
}
