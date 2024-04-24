package railigentxotelreceiver

import (
	"context"

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
	return &railigentXReceiver{
		logger:           params.Logger,
		config:           config,
		nextConsumer:     consumer,
		railigentxClient: gorailigentx.NewClient(config.BaseURL, config.Username, config.Password),
	}, nil
}

func createDefaultConfig() component.Config {
	return &Config{}
}
