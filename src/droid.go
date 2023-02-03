package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/spf13/viper"
)

var (
	config    Config
	statusURL = config.RPCEndpoint + "/status"
	epochURL  = config.LCDEndpoint + "/osmosis/epochs/v1beta1/epochs"
)

type Config struct {
	RPCEndpoint string `mapstructure:"RPC_ENDPOINT"`
	LCDEndpoint string `mapstructure:"LCD_ENDPOINT"`
}

type NodeStatus struct {
	Result Result `json:"result"`
}

type Result struct {
	NodeInfo      NodeInfo      `json:"node_info"`
	SyncInfo      SyncInfo      `json:"sync_info"`
	ValidatorInfo ValidatorInfo `json:"validator_info"`
}

type SyncInfo struct {
	LatestAppHash       string    `json:"latest_app_hash"`
	LatestBlockHash     string    `json:"latest_block_hash"`
	LatestBlockHeight   string    `json:"latest_block_height"`
	LatestBlockTime     time.Time `json:"latest_block_time"`
	EarliestBlockHash   string    `json:"earliest_block_hash"`
	EarliestAppHash     string    `json:"earliest_app_hash"`
	EarliestBlockHeight string    `json:"earliest_block_height"`
	EarliestBlockTime   time.Time `json:"earliest_block_time"`
	CatchingUp          bool      `json:"catching_up"`
}

type NodeInfo struct {
	ID      string `json:"id"`
	Network string `json:"network"`
}

type ValidatorInfo struct {
	Address string `json:"address"`
	PubKey  PubKey `json:"pub_key"`
}

type PubKey struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type Block struct {
	Hash   string    `json:"hash"`
	Height string    `json:"height"`
	Time   time.Time `json:"time"`
}

func GetNodeStatus(statusURL string) NodeStatus {
	resp, err := http.Get(statusURL)
	if err != nil {
		log.Fatalln(err)
	}

	defer resp.Body.Close()
	var nodeStatus NodeStatus
	if err := json.NewDecoder(resp.Body).Decode(&nodeStatus); err != nil {
		log.Fatalln(err)
	}

	return nodeStatus
}

func getLatestBlockFromNodeStatus(status NodeStatus) Block {

	return Block{
		Height: status.Result.SyncInfo.LatestBlockHeight,
		Hash:   status.Result.SyncInfo.LatestBlockHash,
		Time:   status.Result.SyncInfo.LatestBlockTime,
	}
}

// /block handler
func getLatestBlockHandler(w http.ResponseWriter, r *http.Request) {

	status := GetNodeStatus(statusURL)
	latestBlock := getLatestBlockFromNodeStatus(status)

	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	err := enc.Encode(latestBlock)

	if err != nil {
		log.Fatalf("Failed to encode block, %s\n", err)
		io.WriteString(w, "Internal Server Error\n")
	}
}

// /height handler
func getLatestBlockHeightHandler(w http.ResponseWriter, r *http.Request) {

	status := GetNodeStatus(statusURL)
	latestBlock := getLatestBlockFromNodeStatus(status)
	_, _ = io.WriteString(w, latestBlock.Height)
}

// /node_id handler
func getNodeIDHandler(w http.ResponseWriter, r *http.Request) {

	status := GetNodeStatus(statusURL)
	_, _ = io.WriteString(w, status.Result.NodeInfo.ID)
}

// /pub_key handler
func getPubKeyHandler(w http.ResponseWriter, r *http.Request) {
	status := GetNodeStatus(statusURL)

	response := map[string]string{
		"@type": "/cosmos.crypto.ed25519.PubKey",
		"key":   status.Result.ValidatorInfo.PubKey.Value,
	}

	data, _ := json.Marshal(response)
	_, _ = io.WriteString(w, string(data))
}

// global cached next epoch start time
var nextEpochStartTime time.Time

// This variable controls how many minutes after epoch we should expect the node to properly respond
// We assume epoch starts 5 minutes before the calculated epoch
var minutesAfterEpoch time.Duration = 35

func calculateEpoch() time.Time {
	failure := time.Time{}
	// Calculate the epoch from the node
	resp, err := http.Get(epochURL)
	if err != nil {
		log.Fatalln(err)
	}

	defer resp.Body.Close()
	bz, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return failure
	}
	if resp.StatusCode != http.StatusOK {
		fmt.Println(err)
		return failure
	}

	// Unmarshall the epoch json into a map from string to an array of interfaces
	var epoch map[string]interface{}
	err = json.Unmarshal(bz, &epoch)
	if err != nil {
		fmt.Println(err)
		return failure
	}

	// extract the "epoch" array from the epoch map
	epochArray := epoch["epochs"].([]interface{})

	// iterate over the array and find the element of the array that has identifier "day"
	for _, epochObject := range epochArray {
		epochObject := epochObject.(map[string]interface{})
		epochIdentifier := epochObject["identifier"].(string)
		//check that the identifier is "day"
		if epochIdentifier != "day" {
			continue
		}
		// extract the "current_epoch_start_time" from the epochObject and parse it as a date
		epochStartTimeString := epochObject["current_epoch_start_time"].(string)
		epochStartTime, err := time.Parse(time.RFC3339, epochStartTimeString)
		if err != nil {
			fmt.Println(err)
			return failure
		}
		// add 24 hours to the epoch start time
		nextEpochStartTime = epochStartTime.Add(24 * time.Hour)
		// subtract 5 minutes
		nextEpochStartTime = nextEpochStartTime.Add(-5 * time.Minute)
		return nextEpochStartTime
	}
	fmt.Println("No epochs found")
	return failure
}

// Function getEpoch2 which queries the epochURL and parses it as json
func isWithinEpoch(currentTime time.Time) bool {
	// Check if we have a cached epoch
	if nextEpochStartTime.IsZero() || currentTime.After(nextEpochStartTime.Add(minutesAfterEpoch*time.Minute)) {
		// if there is no cache or the epoch has ended, recalculate
		nextEpochStartTime = calculateEpoch()
	}
	// otherwise, use the cached value

	if currentTime.After(nextEpochStartTime) && currentTime.Before(nextEpochStartTime.Add(minutesAfterEpoch*time.Minute)) {
		return true
	}
	return false
}

// /health handler
func healthcheckHandler(w http.ResponseWriter, r *http.Request) {

	status := GetNodeStatus(statusURL)
	latestBlock := getLatestBlockFromNodeStatus(status)

	catchingUp := status.Result.SyncInfo.CatchingUp
	secondsSinceLastBlock := int(time.Now().UTC().Sub(latestBlock.Time).Seconds())

	// To test with fake epoch
	//timsStr := "2023-01-25T13:31:46.131312500Z"
	//t, _ := time.Parse(time.RFC3339Nano, timsStr)

	t := time.Now()
	isEpoch := isWithinEpoch(t)

	healthStatus := http.StatusOK
	healthMessage := "UP"

	if catchingUp || (secondsSinceLastBlock >= 60 && !isEpoch) {
		healthStatus = http.StatusServiceUnavailable
		healthMessage = "DOWN"
	}

	w.WriteHeader(healthStatus)
	_, _ = io.WriteString(w, fmt.Sprintf("%s\nLatest Block %s (Received %d seconds ago)\n", healthMessage, latestBlock.Height, secondsSinceLastBlock))
}

func setNodeEndpoints() error {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./")
	err := viper.ReadInConfig()
	if err != nil {
		return err
	}

	configUnmarshalled := Config{}
	err = viper.Unmarshal(&configUnmarshalled)
	if err != nil {
		return err
	}
	config = configUnmarshalled
	return nil
}

func main() {
	fmt.Println("[ 🤖 droid starting ]")

	// set config & endpoints
	fmt.Println("setting node endpoints...")
	err := setNodeEndpoints()
	if err != nil {
		panic("end points config are not set correctly")
	}
	fmt.Println("RPC end point: ", config.RPCEndpoint)
	fmt.Println("LCD end point: ", config.LCDEndpoint)

	fmt.Printf("listening on port %d...\n", 8080)

	http.HandleFunc("/node_id", getNodeIDHandler)
	http.HandleFunc("/pub_key", getPubKeyHandler)
	http.HandleFunc("/block", getLatestBlockHandler)
	http.HandleFunc("/height", getLatestBlockHeightHandler)
	http.HandleFunc("/health", healthcheckHandler)

	err = http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatalf("Fail to start server, %s\n", err)
	}
}
