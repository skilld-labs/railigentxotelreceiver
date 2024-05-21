package railigentxotelreceiver

import (
	"context"

	"github.com/mitchellh/mapstructure"
	gorailigentx "github.com/skilld-labs/go-railigentx"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/receiver"
)

var (
	typeStr = component.MustNewType("railigentx")
)

// NewFactory creates a factory for tailtracer receiver.
func NewFactory() receiver.Factory {
	return receiver.NewFactory(
		typeStr,
		createDefaultConfig,
		receiver.WithMetrics(createMetricsReceiver, component.StabilityLevelAlpha),
	)
}

func createMetricsReceiver(_ context.Context, params receiver.CreateSettings, baseCfg component.Config, consumer consumer.Metrics) (receiver.Metrics, error) {
	config := baseCfg.(*Config)
	amr, err := createAssetMetricRepo(config)
	if err != nil {
		return nil, err
	}
	return &railigentXReceiver{
		logger:                params.Logger,
		config:                config,
		nextConsumer:          consumer,
		railigentxClient:      gorailigentx.NewClient(config.BaseURL, config.Username, config.Password),
		assetMetricRepository: amr,
	}, nil
}

func createDefaultConfig() component.Config {
	return &Config{}
}

func createAssetMetricRepo(config *Config) (AssetMetricRepository, error) {
	var amr AssetMetricRepository
	var err error
	var amrConfig interface{}
	switch config.AssetMetricRepository.Name {
	case "bbolt":
		amrConfig = NewBboltAssetMetricRepoConfig()
		if err = mapstructure.Decode(config.AssetMetricRepository.Config, amrConfig); err != nil {
			return nil, err
		}
		amr, err = NewBboltAssetMetricRepo(amrConfig)
	default:
		amrConfig = NewInmemAssetMetricRepoConfig()
		if err = mapstructure.Decode(config.AssetMetricRepository.Config, amrConfig); err != nil {
			return nil, err
		}
		amr, err = NewInmemAssetMetricRepo(amrConfig)
	}
	if err != nil {
		return nil, err
	}
	return amr, nil
}
