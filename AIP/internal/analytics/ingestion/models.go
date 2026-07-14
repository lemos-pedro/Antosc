package ingestion

import "time"


type TowerSnapshot struct {

	TowerID string

	Name string

	Status string

	Operator string


	Metrics []MetricFeature

	Events []EventFeature


	CollectedAt time.Time
}



type MetricFeature struct {

	Name string

	Value float64

	Unit string

}



type EventFeature struct {

	Type string

	Severity string

	Message string

}