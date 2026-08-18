package snmp

import "towercore/internal/core/domain"

type MetricDefinition struct {
	OID                string
	Key                string
	Scale              float64
	IgnoreValues       []float64
	ZeroMeansNotTested bool
}

type AlarmRule struct {
	Key        string
	Severity   domain.EventSeverity
	Threshold  float64
	Condition  string
	Message    string
	IgnoreZero bool // se true, valor exatamente 0 nunca dispara este alarme
	// (usado para métricas onde 0 significa "não aplicável/não
	// cablado", não "valor real de falha" — ex.: fase C de
	// mains num site só com 2 fases monitoradas)
}

type Profile struct {
	Vendor  string
	Metrics []MetricDefinition
	Alarms  []AlarmRule
}

// NormalizeSamples converts a map of OID->rawValue into a map of metricKey->value
// using the profile's MetricDefinitions (applies scale, ignores configured
// sentinel values and emits _not_tested markers when configured).
func NormalizeSamples(profile Profile, samples map[string]float64) map[string]float64 {
	normalized := make(map[string]float64)
	for _, md := range profile.Metrics {
		raw, exists := samples[md.OID]
		if !exists {
			continue
		}
		scale := md.Scale
		if scale == 0 {
			scale = 1
		}
		value := raw * scale
		if isIgnoredMetricValue(value, md.IgnoreValues) {
			continue
		}
		normalized[md.Key] = value
		if md.ZeroMeansNotTested && value == 0 {
			normalized[md.Key+"_not_tested"] = 1
		}
	}
	return normalized
}

func isIgnoredMetricValue(value float64, ignored []float64) bool {
	for _, candidate := range ignored {
		if value == candidate {
			return true
		}
	}
	return false
}
