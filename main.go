package main

import (
        "bufio"
        "encoding/json"
        "flag"
        "fmt"
        "io"
        "net"
        "net/http"
        "os"
        "path/filepath"
        "strings"
        "time"
)

const (
        apiURL      = "http://ip-api.com/json/%s"
        rateLimit   = 45
        defaultDelay = 1200 * time.Millisecond
)

const banner = `
  ___ ____ ____   ____                _____                        
 |_ _|  _ \___ \ / ___| ___  ___     |_   _| __ __ _  ___ ___ _ __ 
  | || |_) |__) | |  _ / _ \/ _ \ _____| || '__/ _ \| __/ _ \ '__|
  | ||  __// __/ | |_| |  __/ (_) |_____| || | | (_| | (_|  __/ |   
 |___|_|  |_____|\____|\___|\___/      |_||_|  \__,_|\___\___|_|      
                                                                                                       
IP2Geo Tracer - CIDR Geolocation Tool
`

func showBanner() {
        fmt.Print(banner)
        fmt.Println()
}

type GeoResponse struct {
        Status      string  `json:"status"`
        Country     string  `json:"country"`
        CountryCode string  `json:"countryCode"`
        Region      string  `json:"region"`
        RegionName  string  `json:"regionName"`
        City        string  `json:"city"`
        Zip         string  `json:"zip"`
        Lat         float64 `json:"lat"`
        Lon         float64 `json:"lon"`
        Timezone    string  `json:"timezone"`
        ISP         string  `json:"isp"`
        Org         string  `json:"org"`
        AS          string  `json:"as"`
        Message     string  `json:"message,omitempty"`
}

type OutputJSON struct {
        CIDR        string      `json:"cidr"`
        QueriedIP   string      `json:"queried_ip"`
        Geolocation GeoResponse `json:"geolocation"`
        Timestamp   string      `json:"timestamp"`
        Error       string      `json:"error,omitempty"`
}

func firstIPFromCIDR(cidr string) (string, error) {
        _, ipnet, err := net.ParseCIDR(cidr)
        if err != nil {
                return "", err
        }
        return ipnet.IP.String(), nil
}

func fetchGeo(ip string) (GeoResponse, error) {
        url := fmt.Sprintf(apiURL, ip)
        client := http.Client{Timeout: 10 * time.Second}
        resp, err := client.Get(url)
        if err != nil {
                return GeoResponse{}, err
        }
        defer resp.Body.Close()

        body, err := io.ReadAll(resp.Body)
        if err != nil {
                return GeoResponse{}, err
        }

        var geo GeoResponse
        if err := json.Unmarshal(body, &geo); err != nil {
                return GeoResponse{}, err
        }
        return geo, nil
}

func safeFilename(cidr string) string {
        return strings.ReplaceAll(cidr, "/", "_") + ".json"
}

func showHelp() {
        fmt.Printf(`Usage: %s --input <cidr_list.txt> [options]

Required:
  --input <file>         Text file with one CIDR per line.

Options:
  --outdir <dir>         Directory to save JSON files (default: geo_results)
  --delay <duration>     Delay between API requests (default: 1.2s, min: 500ms)
  --filter-states <list> Comma-separated region names to keep (e.g., 'Maharashtra,Karnataka')
  --filter-output <file> File to write matching CIDRs (default: allowed_cidrs/cidrs_from_naval_states.txt)
  --help                 Show this help message.

Examples:
  %s --input cidrs.txt
  %s --input cidrs.txt --filter-states "Maharashtra,Karnataka,Gujarat,Kerala"
  %s --input cidrs.txt --outdir results --delay 2s

`, os.Args[0], os.Args[0], os.Args[0], os.Args[0])
}

func main() {
        // Define flags
        inputFile := flag.String("input", "", "File with CIDRs (one per line)")
        outDir := flag.String("outdir", "geo_results", "Directory to save JSON files")
        delay := flag.Duration("delay", defaultDelay, "Delay between API requests")
        filterStates := flag.String("filter-states", "", "Comma-separated region names to filter")
        filterOutput := flag.String("filter-output", "allowed_cidrs/cidrs_from_naval_states.txt", "File to save matching CIDRs")
        help := flag.Bool("help", false, "Show help")

        // Custom usage to show our banner and help
        flag.Usage = func() {
                showBanner()
                showHelp()
        }

        flag.Parse()

        if *help {
                flag.Usage()
                os.Exit(0)
        }

        if *inputFile == "" {
                fmt.Fprintf(os.Stderr, "Error: --input is required.\n\n")
                flag.Usage()
                os.Exit(1)
        }

        // Validate delay
        if *delay < 500*time.Millisecond {
                fmt.Fprintf(os.Stderr, "Warning: Delay too short. Setting to minimum 500ms.\n")
                *delay = 500 * time.Millisecond
        }

        // Show banner (non‑help runs)
        showBanner()

        // Parse filter states
        var filterMap map[string]bool
        if *filterStates != "" {
                filterMap = make(map[string]bool)
                for _, s := range strings.Split(*filterStates, ",") {
                        trimmed := strings.TrimSpace(s)
                        if trimmed != "" {
                                filterMap[trimmed] = true
                        }
                }
                // Create output directory for filtered CIDRs
                if err := os.MkdirAll(filepath.Dir(*filterOutput), 0755); err != nil {
                        fmt.Fprintf(os.Stderr, "Error creating filter output directory: %v\n", err)
                        os.Exit(1)
                }
                // Truncate the output file
                if err := os.WriteFile(*filterOutput, []byte{}, 0644); err != nil {
                        fmt.Fprintf(os.Stderr, "Error preparing filter output file: %v\n", err)
                        os.Exit(1)
                }
        }

        // Read CIDRs
        file, err := os.Open(*inputFile)
        if err != nil {
                fmt.Fprintf(os.Stderr, "Error opening input file: %v\n", err)
                os.Exit(1)
        }
        defer file.Close()

        var cidrs []string
        scanner := bufio.NewScanner(file)
        for scanner.Scan() {
                line := strings.TrimSpace(scanner.Text())
                if line != "" {
                        cidrs = append(cidrs, line)
                }
        }
        if err := scanner.Err(); err != nil {
                fmt.Fprintf(os.Stderr, "Error reading file: %v\n", err)
                os.Exit(1)
        }

        fmt.Printf("Processing %d CIDRs...\n", len(cidrs))

        // Ensure output directory exists
        if err := os.MkdirAll(*outDir, 0755); err != nil {
                fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
                os.Exit(1)
        }

        matchedCount := 0
        for i, cidr := range cidrs {
                fmt.Printf("[%d/%d] Looking up %s...\n", i+1, len(cidrs), cidr)

                firstIP, err := firstIPFromCIDR(cidr)
                if err != nil {
                        fmt.Printf("   Invalid CIDR %s: %v\n", cidr, err)
                        continue
                }

                geo, err := fetchGeo(firstIP)
                output := OutputJSON{
                        CIDR:      cidr,
                        QueriedIP: firstIP,
                        Timestamp: time.Now().UTC().Format(time.RFC3339),
                }
                if err != nil {
                        output.Error = err.Error()
                } else {
                        output.Geolocation = geo
                        if geo.Status != "success" {
                                output.Error = geo.Message
                        }
                }

                // Write JSON file
                filename := safeFilename(cidr)
                outPath := filepath.Join(*outDir, filename)
                data, err := json.MarshalIndent(output, "", "  ")
                if err != nil {
                        fmt.Printf("   JSON marshal error: %v\n", err)
                        continue
                }
                if err := os.WriteFile(outPath, data, 0644); err != nil {
                        fmt.Printf("   Write error: %v\n", err)
                        continue
                }
                fmt.Printf("   Saved to %s\n", outPath)

                // Filtering
                if filterMap != nil && output.Geolocation.Status == "success" {
                        region := output.Geolocation.RegionName
                        if filterMap[region] {
                                f, err := os.OpenFile(*filterOutput, os.O_APPEND|os.O_WRONLY, 0644)
                                if err != nil {
                                        fmt.Printf("   Warning: cannot open filter output: %v\n", err)
                                } else {
                                        _, _ = f.WriteString(cidr + "\n")
                                        f.Close()
                                        matchedCount++
                                        fmt.Printf("   [+] Matched filter (%s)\n", region)
                                }
                        }
                }

                // Rate limiting
                if i < len(cidrs)-1 {
                        time.Sleep(*delay)
                }
        }

        if filterMap != nil {
                fmt.Printf("\nFiltering complete. %d CIDRs matched the states and were saved to %s\n", matchedCount, *filterOutput)
        }
}
