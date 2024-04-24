package railigentxotelreceiver

import (
	"errors"
	"time"
)

type Config struct {
	BaseURL        string        `mapstructure:"baseURL"`
	Username       string        `mapstructure:"username"`
	Password       string        `mapstructure:"password"`
	ScrapeInterval time.Duration `mapstructure:"scrapeInterval"`
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
