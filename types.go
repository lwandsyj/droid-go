package main

import (
	"time"
)

func DefaultConfig() Config {
	return Config{
		RPCEndpoint: "http://0.0.0.0:26657",
		LCDEndpoint: "http://0.0.0.0:1317",
	}
}

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
