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

	"github.com/joho/godotenv"
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
	// Load environment variables from .env file
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file:", err)
		return
	}

	// Get the Nautobot API token
	token := os.Getenv("NAUTOBOT_API_TOKEN")
	if token == "" {
		fmt.Println("Error: NAUTOBOT_API_TOKEN environment variable is not set")
		return
	}

	// Get the Nautobot URL
	url := os.Getenv("NAUTOBOT_URL")
	if url == "" {
		fmt.Println("Error: NAUTOBOT_URL environment variable is not set")
		return
	}

	// Read GraphQL query from file
	queryFile := "graphql_queries/query.gql"
	query, err := os.ReadFile(queryFile)
	if err != nil {
		fmt.Println("Error reading query file:", err)
		return
	}

	// Create the payload for the GraphQL query
	payload := map[string]string{
		"query": string(query), // The GraphQL query as a string
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("Error marshalling payload:", err)
		return
	}

	// Create the HTTP POST request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}

	// Set the necessary headers for the API request
	req.Header.Set("Authorization", "Token "+token)
	req.Header.Set("Content-Type", "application/json")

	// Send the request and get the response
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error making request:", err)
		return
	}
	defer resp.Body.Close()

	// Check the HTTP response status
	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		fmt.Printf("HTTP Response Status: %s\n", resp.Status)
		fmt.Printf("Error Response: %s\n", string(body))

		// Handle specific error cases
		if resp.StatusCode == http.StatusUnauthorized {
			fmt.Println("Error: Invalid token provided.")
		} else {
			fmt.Println("Error: An unexpected error occurred.")
		}
		return
	}

	// Read the response body
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return
	}

	// Decode the JSON response into the Response struct
	var response Response
	err = json.Unmarshal(body, &response)
	if err != nil {
		fmt.Println("Error decoding JSON:", err)
		return
	}

	// Ensure that devices exist in the response
	if response.Data.Devices == nil || len(response.Data.Devices) == 0 {
		log.Println("No devices found in response. Check your GraphQL query or Nautobot instance.")
		return
	}

	// Prepare the output JSON structure
	var output []map[string]interface{}

	// Regular expression to remove CIDR from IP addresses
	re := regexp.MustCompile(`/[0-9]+.*`)

	// Iterate through devices and build the output structure
	for _, device := range response.Data.Devices {
		// Skip devices without a valid primary IP or role
		if device.PrimaryIP != nil && device.Role != nil && device.Role.Name != "" {
			// Remove CIDR notation from the IP address
			deviceIP := re.ReplaceAllString(device.PrimaryIP.Address, "")

			// Construct the Prometheus scrape target structure
			entry := map[string]interface{}{
				"targets": []string{deviceIP},
				"labels": map[string]string{
					"__meta_prometheus_job": device.Role.Name,
					"__meta_datacenter":     device.Location.Name,
				},
			}
			output = append(output, entry)
		}
	}

	// Convert the output structure to JSON
	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return
	}

	// Start the HTTP server to serve the JSON data
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonData)
	})

	// Serve the JSON data on port 6645
	fmt.Println("Serving on :6645")
	err = http.ListenAndServe(":6645", nil)
	if err != nil {
		fmt.Println("Error starting server:", err)
	}
}
