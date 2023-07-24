package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var (
	config    Config
	statusURL = config.RPCEndpoint + "/status"
	epochURL  = config.LCDEndpoint + "/osmosis/epochs/v1beta1/epochs"
)

var log = logrus.New()

func GetNodeStatus(statusURL string) NodeStatus {
	var nodeStatus NodeStatus
	var err error

	for i := 0; i < 48; i++ { // Retry for up to 4 minute (48 * 5 seconds)
		resp, err := http.Get(statusURL)
		if err == nil {
			defer resp.Body.Close()
			if err := json.NewDecoder(resp.Body).Decode(&nodeStatus); err == nil {
				return nodeStatus
			}
		}
		if i < 11 {
			log.Info("node connection refused...retrying...")
			// If this is not the last retry, wait for 5 seconds before retrying
			time.Sleep(5 * time.Second)
		}
	}
	log.Info("node refused to connect for 4 minute...exiting program")
	// If all retries fail, return empty node status
	log.Fatalln(err)
	return NodeStatus{}
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
	isEpoch := isWithinEpoch(time.Now())

	var healthStatus int
	var healthMessage string

	if catchingUp || (secondsSinceLastBlock >= 60 && !isEpoch) {
		healthStatus = http.StatusServiceUnavailable
		healthMessage = "DOWN"
	} else {
		healthStatus = http.StatusOK
		healthMessage = "UP"
	}

	w.WriteHeader(healthStatus)
	_, _ = io.WriteString(w, fmt.Sprintf("%s\nLatest Block %s (Received %d seconds ago)\n", healthMessage, latestBlock.Height, secondsSinceLastBlock))

	log.WithFields(logrus.Fields{
		"catchingUp":            catchingUp,
		"secondsSinceLastBlock": secondsSinceLastBlock,
		"isEpoch":               isEpoch,
		"healthStatus":          healthStatus,
		"healthMessage":         healthMessage,
	}).Debug("Healthcheck status")
}

func readConfig() error {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("$HOME/.droid")
	viper.AutomaticEnv()

	if err := viper.BindEnv("rpc_endpoint", "RPC_ENDPOINT"); err != nil {
		return err
	}

	if err := viper.BindEnv("lcd_endpoint", "LCD_ENDPOINT"); err != nil {
		return err
	}

	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("Error reading config file: %s", err)
	}

	config.RPCEndpoint = viper.GetString("rpc_endpoint")
	config.LCDEndpoint = viper.GetString("lcd_endpoint")
	return nil
}

func main() {

	log.Info("🤖 droid is starting..")

	log.Info("Reading configuration.")
	if err := readConfig(); err != nil {
		log.Fatalf("Error reading configuration: %s", err)
	}
	log.Infof("RPC: %s", config.RPCEndpoint)
	log.Infof("LCD: %s", config.LCDEndpoint)
	log.Infof("Listening on port %s...", ":8080")

	statusURL = config.RPCEndpoint + "/status"
	epochURL = config.LCDEndpoint + "/osmosis/epochs/v1beta1/epochs"

	http.HandleFunc("/node_id", getNodeIDHandler)
	http.HandleFunc("/pub_key", getPubKeyHandler)
	http.HandleFunc("/block", getLatestBlockHandler)
	http.HandleFunc("/height", getLatestBlockHeightHandler)
	http.HandleFunc("/health", healthcheckHandler)

	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalf("Fail to start server, %s\n", err)
	}
}
