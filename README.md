# RailigentX Receiver

This project provides an OpenTelemetry receiver that scrapes metrics from the RailigentX API and sends them to a [OpenTelemetry Collector](https://opentelemetry.io/docs/collector/quick-start/).

## Table of Contents

- [Overview](#overview)
- [Features](#features)
- [Installation](#installation)
- [Configuration](#configuration)
- [Logging](#logging)
- [Contributing](#contributing)
- [License](#license)

## Overview

The RailigentX Receiver is a component for the [OpenTelemetry Collector](https://opentelemetry.io/docs/collector/quick-start/) that collects metrics from the RailigentX API.

It scrapes data from RailigentX at specified intervals and transforms the data into OpenTelemetry metrics which are then consumed by the Collector.

## Features

- Scrapes asset metrics such as GPS location, mileage, and speed from RailigentX (more TODO).
- Periodic scraping at user-defined intervals.
- Configurable logging for monitoring and debugging.

## Installation

To build a custom collector that includes the RailigentX receiver, you can use the following `builder-config.yaml` configuration (eg here to scrape railigentx metrics and export them with prometheus):

```yaml
dist:
  name: custom-otelcol
  description: "Custom OpenTelemetry Collector with RailigentX Receiver"
  output_path: ./dist

receivers:
  - gomod:
      github.com/skilld-labs/railigentxotelreceiver v1.0.0

exporters:
  - gomod:
      github.com/open-telemetry/opentelemetry-collector-contrib/exporter/prometheusexporter v0.100.0
```

To build and run the custom collector:

1. Install the OpenTelemetry Collector builder:
   ```bash
   go install go.opentelemetry.io/collector/cmd/builder@latest
   ```
   Click [here](https://github.com/open-telemetry/opentelemetry-collector/tree/main/cmd/builder) for more details

2. Create the custom collector using the builder configuration:
   ```bash
   builder --config builder-config.yaml
   ```

3. Create a configuration file `config.yaml` for the collector [see instructions](#configuration)

3. Run the custom collector:
   ```bash
   ./dist/custom-otelcol --config config.yaml
   ```

## Configuration

The receiver requires a configuration file to specify settings such as the RailigentX API credentials, scrape interval, and logging level. Below is an example configuration file:

```yaml
receivers:
  railigentx:
    baseURL: "https://api.railigentx.com"
    username: yourusername
    password: yourpassword
    scrapeInterval: 10s

exporters:
  prometheus:
    endpoint: "0.0.0.0:9090"
    resource_to_telemetry_conversion:
      enabled: true

service:
  pipelines:
    metrics:
      receivers: [railigentx]
      exporters: [prometheus]
  telemetry:
    logs:
      level: debug
```

### Configuration Options

- `baseURL`: The RailigentX API base URL.
- `username`: Your RailigentX username.
- `password`: Your RailigentX password.
- `scrape_interval`: The interval at which the receiver scrapes metrics from RailigentX (e.g., `60s` for 60 seconds).

## Logging

The receiver uses the `zap` logger for logging. The logging configuration can be adjusted in the configuration file. The logs provide detailed information about the lifecycle of the receiver, the scraping process, and any errors encountered.

## Contributing

Contributions are welcome! Please open an issue or submit a pull request on GitHub. Ensure your code adheres to the project's coding standards and includes appropriate tests.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
