package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

// Structs for JSON response
type Response struct {
	Data struct {
		Devices []Device `json:"devices"`
	} `json:"data"`
}

type Device struct {
	Name      string     `json:"name"`
	Role      *Role      `json:"role"`
	Location  Location   `json:"location"`
	PrimaryIP *IPAddress `json:"primary_ip4"`
}

type Role struct {
	Name string `json:"name"`
}

type Location struct {
	Name string `json:"name"`
}

type IPAddress struct {
	Address string `json:"address"`
}

func main() {
	// Load the .env file
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file:", err)
		return
	}

	// Get the token
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

	// Create the payload
	payload := map[string]string{
		"query": string(query),
	}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("Error marshalling payload:", err)
		return
	}

	// Create the HTTP request
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(payloadBytes))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}

	// Set headers
	req.Header.Set("Authorization", "Token "+token)
	req.Header.Set("Content-Type", "application/json")

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error making request:", err)
		return
	}
	defer resp.Body.Close()

	// Check for HTTP response errors
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

	// Decode the JSON response into the struct
	var response Response
	err = json.Unmarshal(body, &response)
	if err != nil {
		fmt.Println("Error decoding JSON:", err)
		return
	}

	// Validate that the required fields exist
	if response.Data.Devices == nil || len(response.Data.Devices) == 0 {
		log.Println("No devices found in response. Double check your GraphQL and/or Nautobot instance")
		return
	}

	// Output JSON structure
	var output []map[string]interface{}

	// Print the decoded response
	for _, device := range response.Data.Devices {
		// Ensure PrimaryIP and Role are not nil
		if device.PrimaryIP != nil && device.Role != nil && device.Role.Name != "" {
			entry := map[string]interface{}{
				"targets": []string{device.PrimaryIP.Address},
				"labels": map[string]string{
					"__meta_prometheus_job": device.Role.Name,
					"__meta_datacenter":     device.Location.Name,
				},
			}
			output = append(output, entry)
		}
	}

	// Convert to JSON
	jsonData, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		fmt.Println("Error marshalling JSON:", err)
		return
	}

	// Print the JSON
	// Start HTTP server
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(jsonData)
	})

	fmt.Println("Serving on http://localhost:6645")
	err = http.ListenAndServe(":6645", nil)
	if err != nil {
		fmt.Println("Error starting server:", err)
	}

}
