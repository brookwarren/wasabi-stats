package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
)

func main() {
	// Parse command-line flags for access key and secret (optional, can also use args)
	var accessKeyID string
	var secretAccessKey string
	flag.StringVar(&accessKeyID, "access-key", "", "Wasabi Access Key ID")
	flag.StringVar(&secretAccessKey, "secret-key", "", "Wasabi Secret Access Key")
	flag.Parse()

	// If flags not provided, fall back to positional args
	args := flag.Args()
	if accessKeyID == "" && len(args) > 0 {
		accessKeyID = args[0]
	}
	if secretAccessKey == "" && len(args) > 1 {
		secretAccessKey = args[1]
	}

	if accessKeyID == "" || secretAccessKey == "" {
		fmt.Fprintln(os.Stderr, "Error: Access Key ID and Secret Access Key must be provided as arguments or flags")
		os.Exit(1)
	}

	// Make the HTTP request
	url := "https://stats.wasabisys.com/v1/standalone/utilizations?latest=true"
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating request: %v\n", err)
		os.Exit(1)
	}
	req.Header.Set("Authorization", accessKeyID+":"+secretAccessKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error making request: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(os.Stderr, "Error: received status code %d\n", resp.StatusCode)
		os.Exit(1)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading response: %v\n", err)
		os.Exit(1)
	}

	// Parse JSON dynamically
	var data map[string]interface{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing JSON: %v\n", err)
		os.Exit(1)
	}

	records, ok := data["Records"].([]interface{})
	if !ok || len(records) == 0 {
		fmt.Fprintln(os.Stderr, "Error: no records found in response")
		os.Exit(1)
	}

	record, ok := records[0].(map[string]interface{})
	if !ok {
		fmt.Fprintln(os.Stderr, "Error: invalid record format")
		os.Exit(1)
	}

	padded, _ := record["PaddedStorageSizeBytes"].(float64) // Use float64 for safety
	metadata, _ := record["MetadataStorageSizeBytes"].(float64)
	deleted, _ := record["DeletedStorageSizeBytes"].(float64)
	objects, _ := record["NumBillableObjects"].(float64)

	// Calculations
	const tib = 1099511627776.0
	activeStorageTiB := (padded + metadata) / float64(tib)
	deletedStorageTiB := deleted / float64(tib)
	totalObjects := int64(objects) // Cast back if needed

	// Output as JSON
	output := map[string]interface{}{
		"active":  fmt.Sprintf("%.2f", activeStorageTiB),
		"deleted": fmt.Sprintf("%.2f", deletedStorageTiB),
		"objects": totalObjects,
	}

	jsonOutput, err := json.Marshal(output)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling output JSON: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(string(jsonOutput))
}
