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

	// Print the decoded response
	for _, device := range response.Data.Devices {
		fmt.Printf("Device Name: %s\n", device.Name)
		fmt.Printf("Location: %s\n", device.Location.Name)
		if device.Role != nil && device.Role.Name != "" {
			fmt.Printf("Role: %s\n", device.Role.Name)
		}
		if device.PrimaryIP != nil {
			fmt.Printf("Primary IP: %s\n", device.PrimaryIP.Address)
		}
		fmt.Println()
	}
}
