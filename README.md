# ip2geo tracer

A command-line tool that reads a list of CIDR ranges, queries the [ip-api.com](http://ip-api.com) free geolocation API for the first IP of each range, and saves the results as JSON files. Optionally, it can filter and save matching CIDRs based on region names (e.g., Indian states hosting naval bases).

## Features

- Batch processing of CIDR lists from a text file (one CIDR per line).
- Automatic rate limiting (45 requests/minute) to respect the free API tier.
- Saves detailed geolocation data (country, region, city, ISP, AS, etc.) for each CIDR.
- Optional region‑based filtering: write CIDRs that match a comma‑separated list of region names to a separate output file.
- Configurable output directory for JSON files and delay between requests.

## Project Structure

```
ip2geo-tracer/
├── main.go                 # main Go source code
├── README.md               # documentation
├── LICENSE                 # MIT license
├── cidrs.txt               # example input file (one CIDR per line)
├── .gitignore              # optional: ignore geo_results/, allowed_cidrs/, binary
└── (output directories are created at runtime)
    ├── geo_results/        # JSON files for each CIDR
    └── allowed_cidrs/      # filtered CIDR list (if --filter-states used)
```

## Installation

### Prerequisites

- Go 1.16 or later

### Build from source

```bash
git clone https://github.com/yourusername/ip2geo-tracer.git
cd ip2geo-tracer
go build -o ip2geo-tracer main.go
```

The source file is `main.go`. Adjust the name if you renamed it.

## Usage

```bash
./ip2geo-tracer --input <cidr_list.txt> [options]
```

### Required flags

| Flag | Description |
|------|-------------|
| `--input` | Path to a text file containing CIDR ranges, one per line. |

### Optional flags

| Flag | Default | Description |
|------|---------|-------------|
| `--outdir` | `geo_results` | Directory where individual JSON files will be saved. |
| `--delay` | `1.2s` | Delay between API requests (e.g., `1.5s`, `800ms`). |
| `--filter-states` | (empty) | Comma‑separated list of region names to filter (e.g., `Maharashtra,Karnataka`). |
| `--filter-output` | `allowed_cidrs/cidrs_from_naval_states.txt` | File to append matching CIDRs when `--filter-states` is used. |

## Examples

### 1. Basic usage – no filtering

Process `cidrs.txt` and save all JSON files into `./geo_results`:

```bash
./ip2geo-tracer --input cidrs.txt
```

### 2. Custom output directory and slower rate limit

```bash
./ip2geo-tracer --input cidrs.txt --outdir my_data --delay 2s
```

### 3. Filter CIDRs that belong to specific Indian states

Extract only CIDRs located in Maharashtra, Karnataka, Gujarat, or Kerala and write them to `naval_cidrs.txt`:

```bash
./ip2geo-tracer --input cidrs.txt --filter-states "Maharashtra,Karnataka,Gujarat,Kerala" --filter-output naval_cidrs.txt
```

All JSON files are still saved normally (in `geo_results` by default). The filtered CIDRs are appended to the specified file, one per line.

### 4. Combine all options

```bash
./ip2geo-tracer --input cidrs.txt --outdir results --delay 1.5s --filter-states "Maharashtra,Karnataka" --filter-output allowed/naval.txt
```

## Output Format

### JSON file (per CIDR)

Each JSON file (named `cidr_netmask.json`, e.g., `1.6.0.0_15.json`) contains:

```json
{
  "cidr": "1.6.0.0/15",
  "queried_ip": "1.6.0.0",
  "geolocation": {
    "status": "success",
    "country": "India",
    "countryCode": "IN",
    "region": "MH",
    "regionName": "Maharashtra",
    "city": "Navi Mumbai",
    "zip": "400615",
    "lat": 19.17,
    "lon": 73.0014,
    "timezone": "Asia/Kolkata",
    "isp": "Sify Limited",
    "org": "Sify space",
    "as": "AS9583 Sify Limited"
  },
  "timestamp": "2026-05-26T12:28:19Z",
  "error": ""
}
```

If the API request fails, the `error` field will contain a description.

### Filter output file

When `--filter-states` is provided, each matching CIDR is appended (one per line) to the file specified by `--filter-output`. Example content:

```
1.6.0.0/15
103.170.36.0/22
119.31.171.20/32
```

## Rate Limiting Notes

- The free ip-api.com tier allows **45 requests per minute** (from a single IP).
- The tool defaults to a 1.2‑second delay between requests, which is safe.
- If you have many CIDRs, the process will take approximately `(number_of_CIDRs) * 1.2 seconds`.
- For large lists, consider increasing the delay or running the tool overnight.

## Error Handling

- Invalid CIDR lines are skipped with a warning.
- API timeouts or failures are captured in the JSON `error` field; processing continues.
- The tool exits immediately if the input file cannot be read or the output directory cannot be created.

## License

MIT License – see [LICENSE](LICENSE) file for details.
