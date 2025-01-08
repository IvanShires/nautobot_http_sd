# Nautobot HTTP Service Discovery

This project is a Go-based HTTP service that integrates with Nautobot to fetch device information using a GraphQL query, process the data, and expose it as a JSON endpoint for Prometheus to scrape as an HTTP SD endpoint.

## Features

- Fetches device data from Nautobot using a GraphQL API.
- Cleans up IP addresses by removing CIDR notation.
- Formats the data into Prometheus scrape target format.
- Serves the data via an HTTP endpoint on port `6645` (default).
- Easy integration with Prometheus for dynamic HTTP-based service discovery.

## Prerequisites

- Docker installed on the target machine.
- Nautobot instance with the GraphQL API enabled.
- A valid Nautobot API token with read permissions.

## Install / Usage

Clone the repository:
   
```
git clone https://github.com/IvanShires/nautobot_http_sd.git
cd nautobot_http_sd
docker build -t nautobot_http_sd .
docker run -d \
--name=nautobot_http_sd \
-e NAUTOBOT_API_TOKEN=your_token_here \
-e NAUTOBOT_API_TOKEN=your_token_here \
-e NAUTOBOT_URL=https://nautobot.domain.tld/api/graphql/ \
-p 6645:6645 \
nautobot_http_sd
```
    
I use this with Prometheus, so here is my `prometheus.yml` configuration, for Blackbox Exporter:

```
# cat prometheus.yml 
scrape_configs:
  - job_name: 'blackbox_icmp'
    metrics_path: /probe
    params:
      module: [icmp] # Use the ICMP module for ping-like probing
    http_sd_configs:
      - url: https://nautobot_http_sd.domain.tld:6645 # HTTP Service Discovery endpoint
        refresh_interval: 5m
    relabel_configs:
      - source_labels: [__address__]
        target_label: __param_target # Pass target IP as the `target` parameter to Blackbox Exporter
      - source_labels: [__param_target]
        target_label: instance # Set the `instance` label to the target IP for better labeling
      - target_label: __address__
        replacement: https://prometheus.domain.tld:9115 # Replace with the Blackbox Exporter's address
```
