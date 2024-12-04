package main

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"encoding/json"
	"bytes"
	"time"
	"os"
	"bufio"
)
import("github.com/common-nighthawk/go-figure")

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	Expiry      string `json:"expiry"`
}

type Device struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Pubkey    string `json:"pubkey"`
	HijackDNS bool   `json:"hijack_dns"`
	Created   string `json:"created"`
	IPv4      string `json:"ipv4_address"`
	IPv6      string `json:"ipv6_address"`
	Ports     []int  `json:"ports"`
}

var nameToPubkey = make(map[string]string)

func getToken(accnumber int) string {
fmt.Println("Getting access token")
client := &http.Client{}

accnumberstr := strconv.Itoa(accnumber)
fmt.Println("Account Number:", accnumberstr)

// Create JSON payload with account_number
	payload := map[string]string{"account_number": accnumberstr}
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		fmt.Println("Error creating JSON payload:", err)
		return ""
	}

req, err := http.NewRequest("POST", "https://api.mullvad.net/auth/v1/token", bytes.NewBuffer(jsonPayload))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return ""
	}

	// Add headers
	req.Header.Set("Content-Type", "application/json") // JSON format header

	// Execute the request
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error executing request:", err)
		return ""
	}
	defer resp.Body.Close()

	// Read Response Body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return ""
	}

	// Parse the JSON response
	var tokenResp TokenResponse
	err = json.Unmarshal(respBody, &tokenResp)
	if err != nil {
		fmt.Println("Error parsing JSON:", err)
		return ""
	}
	fmt.Println("Successfully obtained access token:", tokenResp.AccessToken)
	return tokenResp.AccessToken
}

func getDevices(accessToken string) {
	client := &http.Client{}

	req, err := http.NewRequest("GET", "https://api.mullvad.net/accounts/v1/devices", nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accepts", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	// Execute the request
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error executing request:", err)
		return
	}
	defer resp.Body.Close()

	// Read Response Body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return
	}

	// Parse JSON into a slice of Device structs
	var devices []Device
	err = json.Unmarshal(respBody, &devices)
	if err != nil {
		fmt.Println("Error parsing JSON:", err)
		return
	}

	// Create a map of name:pubkey pairs for fast lookups
	for _, device := range devices {
		nameToPubkey[device.Name] = device.Pubkey
	}
} 

func deleteDevices(pubkey string, accessToken string) {
client := &http.Client{}

payload := map[string]string{"pubkey": pubkey}
jsonPayload, err := json.Marshal(payload)
if err != nil {
	fmt.Println("Error creating JSON payload:", err)
	return
}

	req, err := http.NewRequest("POST", "https://api.mullvad.net/www/wg-pubkeys/revoke/", bytes.NewBuffer(jsonPayload))
	if err != nil {
		fmt.Println("Error creating request:", err)
		return
	}

	// Add headers
	req.Header.Add("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", "application/json")

	// Execute the request
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error executing request:", err)
		return
	}
	defer resp.Body.Close()
}

func getPubkeyByName(name string, nameToPubkey map[string]string) (string) {
	// Lookup the pubkey for the given name
	pubkey := nameToPubkey[name]
	return pubkey
}

func getAllPubkeys() []string {
	var pubkeys []string
	// Iterate over the global map and append pubkeys to the slice
	for _, pubkey := range nameToPubkey {
		pubkeys = append(pubkeys, pubkey)
	}
	return pubkeys
}

func loadWantedList(filePath string) (map[string]struct{}, error) {
	// Open the file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("error opening file: %w", err)
	}
	defer file.Close()

	// Create the wanted list map
	wantedList := make(map[string]struct{})

	// Read the file line by line
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if line != "" { // Ignore empty lines
			wantedList[line] = struct{}{}
		}
	}

	// Check for errors while reading the file
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading file: %w", err)
	}

	return wantedList, nil
}


func main() {
myFigure := figure.NewColorFigure("Mullvad Auto Remove", "","green", true)
myFigure.Print()

var accountnumber int
fmt.Println("Enter your account number:")
fmt.Scanln(&accountnumber)

for (1==1) {
getDevices(getToken(accountnumber))

// File path to read from
filePath := "authorised_devices.txt"

// Load the wanted list map
wantedList, err := loadWantedList(filePath)
if err != nil {
	fmt.Println("Error:", err)
	return
}

// Iterate over the nameToPubkey map and check if the name exists in the wanted list
for name, pubkey := range nameToPubkey {
	if _, found := wantedList[name]; !found {
		// Name is NOT in the wanted list, so handle the removal
		fmt.Printf("Found imposter %s with pubkey: %s\nRemoving now...\n", name, pubkey)
		time.Sleep(3 * time.Second) // Delay as to not get ratelimited

		// Call the deletion function
		deleteDevices(pubkey, getToken(accountnumber))
	}
}

// Check for imposters every 5 mins
fmt.Println("All imposters removed!")
time.Sleep(300 * time.Second)
}
}

