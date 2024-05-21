package railigentxotelreceiver

import "time"

type AssetMetric struct {
	Asset  string
	Metric string
}

func (am *AssetMetric) Signature() string {
	return am.Asset + "." + am.Metric
}

type AssetMetricRepository interface {
	Store(*AssetMetric, time.Time) error
	Get(*AssetMetric) (time.Time, bool, error)
}
