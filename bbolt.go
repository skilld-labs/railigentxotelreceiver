package railigentxotelreceiver

import (
	"fmt"
	"time"

	"go.etcd.io/bbolt"
)

type bboltAssetMetricRepo struct {
	db *bbolt.DB
}

type bboltAssetMetricRepoConfig struct {
	DBPath string `mapstructure:"db_path"`
}

func NewBboltAssetMetricRepoConfig() interface{} {
	return &bboltAssetMetricRepoConfig{}
}

func NewBboltAssetMetricRepo(config interface{}) (AssetMetricRepository, error) {
	cfg := config.(*bboltAssetMetricRepoConfig)
	db, err := bbolt.Open(cfg.DBPath, 0600, nil)
	if err != nil {
		return nil, err
	}
	db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("asset_metrics"))
		if err != nil {
			return fmt.Errorf("create bucket: %s", err)
		}
		return nil
	})
	return &bboltAssetMetricRepo{
		db: db,
	}, nil
}

func (r *bboltAssetMetricRepo) Store(am *AssetMetric, ts time.Time) error {
	return r.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("asset_metrics"))
		err := b.Put([]byte(am.Signature()), []byte(ts.Format(time.RFC3339)))
		return err
	})
}

func (r *bboltAssetMetricRepo) Get(am *AssetMetric) (time.Time, bool, error) {
	var ts time.Time
	var exists bool
	var err error
	err = r.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("asset_metrics"))
		v := b.Get([]byte(am.Signature()))
		if string(v) == "" {
			return nil
		}
		exists = true
		ts, err = time.Parse(time.RFC3339, string(v))
		return err
	})
	return ts, exists, err
}
