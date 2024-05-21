package railigentxotelreceiver

import (
	"context"
	"fmt"
	"time"

	"github.com/skilld-labs/go-railigentx"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.uber.org/zap"
)

type railigentXReceiver struct {
	host         component.Host
	cancel       context.CancelFunc
	logger       *zap.Logger
	config       *Config
	nextConsumer consumer.Metrics

	railigentxClient      *railigentx.Client
	assetMetricRepository AssetMetricRepository
}

func (receiver *railigentXReceiver) Start(ctx context.Context, host component.Host) error {
	receiver.logger.Debug("Starting collector")
	receiver.host = host
	ctx, receiver.cancel = context.WithCancel(ctx)
	go receiver.scrapeMetricsLoop(ctx)
	return nil
}

func (receiver *railigentXReceiver) Shutdown(ctx context.Context) error {
	receiver.logger.Info("Shutting down collector")
	receiver.cancel()
	return nil
}

func (receiver *railigentXReceiver) scrapeMetricsLoop(ctx context.Context) {
	ticker := time.NewTicker(receiver.config.ScrapeInterval)
	defer ticker.Stop()
	receiver.logger.Info("Started metrics scraping loop", zap.Duration("interval", receiver.config.ScrapeInterval))

	for {
		receiver.logger.Info("Scraping metrics")
		metricsScrape, err := newMetricsScrape(&metricsScrapeConfig{
			logger:                receiver.logger.Named("metrics_scrape"),
			railigentxClient:      receiver.railigentxClient,
			scrapeInterval:        receiver.config.ScrapeInterval,
			assetMetricRepository: receiver.assetMetricRepository,
		})
		if err != nil {
			receiver.logger.Error("Error while creating new metrics scrape", zap.Error(err))
			continue
		}

		metrics, err := metricsScrape.generateMetrics()
		if err != nil {
			receiver.logger.Error("Error while generating metrics", zap.Error(err))
		}

		err = receiver.nextConsumer.ConsumeMetrics(ctx, metrics)
		if err != nil {
			receiver.logger.Error("Error consuming metrics", zap.Error(err))
		} else {
			receiver.logger.Info("Metrics successfully consumed")
		}
		select {
		case <-ticker.C:
			continue
		case <-ctx.Done():
			receiver.logger.Info("Metrics scraping loop stopped")
			return
		}
	}
}

type metricsScrape struct {
	logger                *zap.Logger
	railigentxClient      *railigentx.Client
	scrapeInterval        time.Duration
	assetMetricRepository AssetMetricRepository
}

type metricsScrapeConfig struct {
	logger                *zap.Logger
	railigentxClient      *railigentx.Client
	scrapeInterval        time.Duration
	assetMetricRepository AssetMetricRepository
}

func newMetricsScrape(cfg *metricsScrapeConfig) (*metricsScrape, error) {
	cfg.logger.Info("Creating new Metrics Scrape instance")
	return &metricsScrape{
		logger:                cfg.logger,
		railigentxClient:      cfg.railigentxClient,
		scrapeInterval:        cfg.scrapeInterval,
		assetMetricRepository: cfg.assetMetricRepository,
	}, nil
}

func (scrape *metricsScrape) generateMetrics() (pmetric.Metrics, error) {
	scrape.logger.Info("Scraping metrics")
	metrics := pmetric.NewMetrics()

	scrapeResourceMetric := metrics.ResourceMetrics().AppendEmpty()
	scrapeInstScope := scrapeResourceMetric.ScopeMetrics().AppendEmpty()

	scrapeStateMetric := scrapeInstScope.Metrics().AppendEmpty()
	scrapeStateMetric.SetName("scrape_state")
	scrapeStateMetric.SetDescription(fmt.Sprintf("The state of the scrape, %d stands for succeed scrape, %d for incomplete scrape and %d for failed scrape", ScrapeSucceeded, ScrapeIncomplete, ScrapeFailed))
	scrapeStateMetric.SetEmptyGauge()

	scrapeState := ScrapeSucceeded
	defer func() {
		scrapeStateMetric.Gauge().DataPoints().AppendEmpty().SetIntValue(int64(scrapeState))
	}()

	observations, err := scrape.scrapeLastAssetsObservations()
	if err != nil {
		scrapeState = ScrapeFailed
		return metrics, fmt.Errorf("Error while scraping observations: %w", err)
	}

	for _, fleetObservations := range observations.FleetsObservations {
		if !fleetObservations.Observed {
			scrape.logger.Error("Fleet has not be observed, scraping is incomplete", zap.String("fleet", fleetObservations.Fleet))
			scrapeState = ScrapeIncomplete
		}
		for _, asset := range fleetObservations.Observations {
			if err := scrape.collectAssetMetrics(metrics, fleetObservations.Fleet, asset); err != nil {
				scrape.logger.Error("Error collecting asset metrics", zap.Error(err))
			}
		}
	}

	scrape.logger.Info("Metrics scraping complete")
	return metrics, nil
}

func (scrape *metricsScrape) collectAssetMetrics(metrics pmetric.Metrics, fleet string, asset railigentx.Asset) error {
	scrape.logger.Debug("Collecting metrics for asset", zap.String("fleet", fleet), zap.String("asset_id", asset.ID))

	resourceMetric := metrics.ResourceMetrics().AppendEmpty()
	assetResource := resourceMetric.Resource()
	assetAttrs := assetResource.Attributes()
	assetAttrs.PutStr("asset.fleet_id", fleet)
	assetAttrs.PutStr("asset.id", asset.ID)
	if asset.Features.UIC != nil {
		assetAttrs.PutStr("asset.uic", asset.Features.UIC.Value)
	}
	if asset.Features.Trip != nil {
		assetAttrs.PutStr("asset.trip_id", asset.Features.Trip.Value.TripID)
	}
	assetInstScope := resourceMetric.ScopeMetrics().AppendEmpty()

	if asset.Features.GPS != nil {
		am := &AssetMetric{Asset: asset.ID, Metric: "asset_gps_latitude"}
		oldTs, exists, err := scrape.assetMetricRepository.Get(am)
		if err != nil {
			scrape.logger.Error("error while accessing asset metric repository", zap.String("asset_id", asset.ID), zap.Error(err))
		} else {
			newTs := time.UnixMilli(asset.Features.GPS.Timestamp)
			if !exists || !newTs.Add(-scrape.scrapeInterval).Before(oldTs) {
				scrape.assetMetricRepository.Store(am, newTs)
				latitudeMetric := assetInstScope.Metrics().AppendEmpty()
				latitudeMetric.SetName("asset_gps_latitude")
				latitudeMetric.SetDescription("The asset GPS latitude, in degree of arc")
				latitudeMetric.SetUnit("deg")
				latitudeMetric.SetEmptyGauge()
				point := latitudeMetric.Gauge().DataPoints().AppendEmpty()
				point.SetTimestamp(pcommon.NewTimestampFromTime(newTs))
				point.SetDoubleValue(asset.Features.GPS.Position.Latitude)
			}
		}

		am = &AssetMetric{Asset: asset.ID, Metric: "asset_gps_longitude"}
		oldTs, exists, err = scrape.assetMetricRepository.Get(am)
		if err != nil {
			scrape.logger.Error("error while accessing asset metric repository", zap.String("asset_id", asset.ID), zap.Error(err))
		} else {
			newTs := time.UnixMilli(asset.Features.GPS.Timestamp)
			if !exists || !newTs.Add(-scrape.scrapeInterval).Before(oldTs) {
				scrape.assetMetricRepository.Store(am, newTs)
				longitudeMetric := assetInstScope.Metrics().AppendEmpty()
				longitudeMetric.SetName("asset_gps_longitude")
				longitudeMetric.SetDescription("The asset GPS longitude, in degree of arc")
				longitudeMetric.SetUnit("deg")
				longitudeMetric.SetEmptyGauge()
				point := longitudeMetric.Gauge().DataPoints().AppendEmpty()
				point.SetTimestamp(pcommon.NewTimestampFromTime(newTs))
				point.SetDoubleValue(asset.Features.GPS.Position.Longitude)
			}
		}
	}

	if asset.Features.Mileage != nil {
		am := &AssetMetric{Asset: asset.ID, Metric: "asset_mileage"}
		oldTs, exists, err := scrape.assetMetricRepository.Get(am)
		if err != nil {
			scrape.logger.Error("error while accessing asset metric repository", zap.String("asset_id", asset.ID), zap.Error(err))
		} else {
			newTs := time.UnixMilli(asset.Features.Mileage.Timestamp)
			if !exists || !newTs.Add(-scrape.scrapeInterval).Before(oldTs) {
				scrape.assetMetricRepository.Store(am, newTs)
				mileageMetric := assetInstScope.Metrics().AppendEmpty()
				mileageMetric.SetName("asset_mileage")
				mileageMetric.SetUnit("km")
				mileageMetric.SetDescription("The asset mileage, in kilometers.")
				mileageMetric.SetEmptyGauge()
				point := mileageMetric.Gauge().DataPoints().AppendEmpty()
				point.SetTimestamp(pcommon.NewTimestampFromTime(newTs))
				point.SetDoubleValue(asset.Features.Mileage.Value)
			}
		}
	}

	if asset.Features.Speed != nil {
		am := &AssetMetric{Asset: asset.ID, Metric: "asset_speed"}
		oldTs, exists, err := scrape.assetMetricRepository.Get(am)
		if err != nil {
			scrape.logger.Error("error while accessing asset metric repository", zap.String("asset_id", asset.ID), zap.Error(err))
		} else {
			newTs := time.UnixMilli(asset.Features.Speed.Timestamp)
			if !exists || !newTs.Add(-scrape.scrapeInterval).Before(oldTs) {
				scrape.assetMetricRepository.Store(am, newTs)
				speedMetric := assetInstScope.Metrics().AppendEmpty()
				speedMetric.SetName("asset_speed")
				speedMetric.SetUnit("km/h")
				speedMetric.SetDescription("The asset speed, in kilometers per hour.")
				speedMetric.SetEmptyGauge()
				point := speedMetric.Gauge().DataPoints().AppendEmpty()
				point.SetTimestamp(pcommon.NewTimestampFromTime(newTs))
				point.SetDoubleValue(asset.Features.Speed.Value)
			}
		}
	}
	return nil
}

type ScrapeState int

const (
	ScrapeSucceeded ScrapeState = iota
	ScrapeFailed
	ScrapeIncomplete
)

// ScrapedData holds data scraped from RailigentX
type ScrapedData struct {
	ObservedAt         time.Time
	FleetsObservations []FleetObservations
}

// FleetObservations holds observations for a fleet
type FleetObservations struct {
	Fleet        string
	Observed     bool
	Observations []railigentx.Asset
}

// scrapeLastAssetsObservations scrapes the latest assets observations from RailigentX
func (scrape *metricsScrape) scrapeLastAssetsObservations() (*ScrapedData, error) {
	scrape.logger.Debug("Starting scrape of latest observations")
	scrapedData := &ScrapedData{
		ObservedAt: time.Now(),
	}

	fleets, err := scrape.railigentxClient.ListFleets()
	if err != nil {
		scrape.logger.Error("Error while listing fleets", zap.Error(err))
		return scrapedData, fmt.Errorf("Failed to list fleets: %v", err)
	}
	for _, fleet := range fleets.Data {
		scrape.logger.Debug("Scraping assets for fleet", zap.String("fleet_id", fleet.ID))
		fleetObs := FleetObservations{
			Fleet:    fleet.ID,
			Observed: false,
		}
		assets, err := scrape.railigentxClient.ListAssets(fleet.ID)
		if err != nil {
			scrape.logger.Error("Error while listing assets", zap.String("fleet", fleet.ID), zap.Error(err))
		} else {
			fleetObs.Observations = assets.Data
			fleetObs.Observed = true
			scrape.logger.Debug("Observations information", zap.String("fleet", fleet.ID), zap.Int("assets_count", len(assets.Data)))
		}
		scrapedData.FleetsObservations = append(scrapedData.FleetsObservations, fleetObs)
	}
	scrape.logger.Info("Scraping complete")
	return scrapedData, nil
}
