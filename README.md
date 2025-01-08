# Nautobot HTTP Service Discovery

This project is a Go-based HTTP service that integrates with Nautobot to fetch device information using a GraphQL query, process the data, and expose it as a JSON endpoint for Prometheus to scrape it to use as an HTTP SD endpoint

## Features

- Fetches device data from Nautobot using a GraphQL API.
- Cleans up IP addresses by removing CIDR notation.
- Formats the data into Prometheus scrape target format.
- Serves the data via an HTTP endpoint on port `6645` (default).