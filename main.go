package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
)

// Structs for decoding the JSON response
type Response struct {
	Data struct {
		Devices []Device `json:"devices"` // List of devices in the response
	} `json:"data"`
}

type Device struct {
	Name      string     `json:"name"`        // Device name
	Role      *Role      `json:"role"`        // Device role
	Location  Location   `json:"location"`    // Device location
	PrimaryIP *IPAddress `json:"primary_ip4"` // Primary IP address of the device
}

type Role struct {
	Name string `json:"name"` // Role name
}

type Location struct {
	Name string `json:"name"` // Location name
}

type IPAddress struct {
	Address string `json:"address"` // IP address with CIDR
}

func main() {
	// Load environment variables for API token and URL
	token := os.Getenv("NAUTOBOT_API_TOKEN")
	url := os.Getenv("NAUTOBOT_URL")

	if token == "" || url == "" {
		log.Fatal("Error: Ensure NAUTOBOT_API_TOKEN and NAUTOBOT_URL are set in the environment")
	}

	// Read all GraphQL query files from the directory
	queryDir := "graphql_queries/"
	entries, err := os.ReadDir(queryDir)
	if err != nil {
		log.Fatalf("Error reading directory %s: %v", queryDir, err)
	}

	// Prepare output for Prometheus scrape targets
	var output []map[string]interface{}
	cidrRegex := regexp.MustCompile(`/[0-9]+.*`) // Regex to remove CIDR notation
	gqlFileRegex := regexp.MustCompile(`\.gql$`) // Regex to strip ".gql" extension

	for _, entry := range entries {
		queryFile := queryDir + entry.Name()

		// Skip non-regular files
		if entry.IsDir() || !gqlFileRegex.MatchString(entry.Name()) {
			continue
		}

		// Read the GraphQL query from the file
		query, err := os.ReadFile(queryFile)
		if err != nil {
			log.Printf("Error reading file %s: %v", queryFile, err)
			continue
		}

		// Send GraphQL query and process the response
		if err := processQuery(url, token, string(query), gqlFileRegex.ReplaceAllString(entry.Name(), ""), &output, cidrRegex); err != nil {
			log.Printf("Error processing query from file %s: %v", queryFile, err)
		}
	}

	// Serve the output as JSON on a local HTTP server
	startServer(output)
}

// processQuery sends a GraphQL query and processes the response
func processQuery(url, token, query, jobName string, output *[]map[string]interface{}, cidrRegex *regexp.Regexp) error {
	// Prepare GraphQL payload
	payloadBytes, err := json.Marshal(map[string]string{"query": query})
	if err != nil {
		return fmt.Errorf("error marshalling payload: %v", err)
	}

	// Create and send the HTTP POST request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return fmt.Errorf("error creating request: %v", err)
	}
	req.Header.Set("Authorization", "Token "+token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error sending request: %v", err)
	}
	defer resp.Body.Close()

	// Handle non-OK response status
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return fmt.Errorf("unexpected HTTP response: %s, body: %s", resp.Status, string(body))
	}

	// Parse the JSON response
	var response Response
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return fmt.Errorf("error decoding response JSON: %v", err)
	}

	// Ensure the response contains devices
	if len(response.Data.Devices) == 0 {
		log.Println("No devices found in response")
		return nil
	}

	// Process each device in the response
	for _, device := range response.Data.Devices {
		if device.PrimaryIP != nil && device.Role != nil && device.Role.Name != "" {
			deviceIP := cidrRegex.ReplaceAllString(device.PrimaryIP.Address, "") // Strip CIDR
			entry := map[string]interface{}{
				"targets": []string{deviceIP},
				"labels": map[string]string{
					"__meta_prometheus_job": jobName,
					"__meta_datacenter":     device.Location.Name,
				},
			}
			*output = append(*output, entry)
		}
	}
	return nil
}

// startServer serves the JSON output on a local HTTP server
func startServer(output []map[string]interface{}) {
	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		log.Fatalf("Error marshalling JSON: %v", err)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonData)
	})

	port := ":6645"
	log.Printf("Serving on %s", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatalf("Error starting server: %v", err)
	}
}
