package railigentxotelreceiver

import "time"

type inmemAssetMetricRepo struct {
	assetMetricsByTimestamp map[string]time.Time
}

func NewInmemAssetMetricRepoConfig() interface{} {
	return nil
}

func NewInmemAssetMetricRepo(config interface{}) (AssetMetricRepository, error) {
	return &inmemAssetMetricRepo{assetMetricsByTimestamp: make(map[string]time.Time)}, nil
}

func (r *inmemAssetMetricRepo) Config() interface{} {
	return nil
}

func (r *inmemAssetMetricRepo) Store(am *AssetMetric, ts time.Time) error {
	r.assetMetricsByTimestamp[am.Signature()] = ts
	return nil
}

func (r *inmemAssetMetricRepo) Get(am *AssetMetric) (time.Time, bool, error) {
	ts, exists := r.assetMetricsByTimestamp[am.Signature()]
	return ts, exists, nil
}
