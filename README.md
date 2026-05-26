# CIDR Geolocation Tool

A command-line tool that reads a list of CIDR ranges, queries the [ip-api.com](http://ip-api.com) free geolocation API for the first IP of each range, and saves the results as JSON files. Optionally, it can filter and save matching CIDRs based on region names (e.g., Indian states hosting naval bases).

## Features

- Batch processing of CIDR lists from a text file (one CIDR per line).
- Automatic rate limiting (45 requests/minute) to respect the free API tier.
- Saves detailed geolocation data (country, region, city, ISP, AS, etc.) for each CIDR.
- Optional region‑based filtering: write CIDRs that match a comma‑separated list of region names to a separate output file.
- Configurable output directory for JSON files and delay between requests.

## Installation

### Prerequisites

- Go 1.16 or later

### Build from source

```bash
git clone <repository-url>   # if applicable
cd goIPtracer
go build -o cidr_geo cidr_geo.go
