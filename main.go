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

func main() {
        inputFile := flag.String("input", "", "File with CIDRs (one per line)")
        outDir := flag.String("outdir", "geo_results", "Directory to save JSON files")
        delay := flag.Duration("delay", defaultDelay, "Delay between API requests")
        filterStates := flag.String("filter-states", "", "Comma-separated region names to filter (e.g. 'Maharashtra,Karnataka')")
        filterOutput := flag.String("filter-output", "allowed_cidrs/cidrs_from_naval_states.txt", "File to save matching CIDRs (ignored if --filter-states is empty)")
        flag.Parse()

        if *inputFile == "" {
                fmt.Fprintf(os.Stderr, "Usage: %s --input <cidr_list.txt> [--outdir <dir>] [--delay 1.2s] [--filter-states 'State1,State2'] [--filter-output <file>]\n", os.Args[0])
                os.Exit(1)
        }

        // Parse filter states if provided
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

        // Read CIDRs from file
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

                // Filtering: if regionName matches and no error
                if filterMap != nil && output.Geolocation.Status == "success" {
                        region := output.Geolocation.RegionName
                        if filterMap[region] {
                                // Append CIDR to filter output file
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
