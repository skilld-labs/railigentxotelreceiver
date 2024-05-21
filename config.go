package railigentxotelreceiver

import (
	"errors"
	"time"
)

type Config struct {
	BaseURL               string                      `mapstructure:"base_url"`
	Username              string                      `mapstructure:"username"`
	Password              string                      `mapstructure:"password"`
	ScrapeInterval        time.Duration               `mapstructure:"scrape_interval"`
	AssetMetricRepository AssetMetricRepositoryConfig `mapstructure:"asset_metric_repository"`
}

type AssetMetricRepositoryConfig struct {
	Name   string      `mapstructure:"name"`
	Config interface{} `mapstructure:"config"`
}

func (cfg *Config) Validate() error {
	if cfg.BaseURL == "" {
		return errors.New("base url is invalid")
	}
	if cfg.Username == "" {
		return errors.New("username is invalid")
	}
	if cfg.Password == "" {
		return errors.New("password is invalid")
	}
	return nil
}
