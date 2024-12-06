package main

import (
	"encoding/json"
	"os"
	"xelis-mining-proxy/log"
)

var PoolProtocol string = "xatum"

type Config struct {
	WalletAddress   string `json:"wallet"`
	PoolUrl         string `json:"pool_url"`
	PoolProtocol    string `json:"pool_protocol"`
	GetworkBindPort uint16 `json:"getwork_bind_port"`
	StratumBindPort uint16 `json:"stratum_bind_port"`
	Debug           bool   `json:"debug"`
}

// 5210: Getwork

var Cfg = Config{
	Debug:           false,
	WalletAddress:   "YOUR_WALLET_ADDRESS",
	PoolUrl:         "127.0.0.1:8080",
	PoolProtocol:    "auto",
	StratumBindPort: 5209,
	GetworkBindPort: 5210,
}

func init() {
	loadCfg()
}

func loadCfg() {
	data, err := os.ReadFile(path() + "/config.json")

	if err != nil {
		log.Warn("failed to open configuration:", err)
		saveCfg()
		return
	}

	err = json.Unmarshal(data, &Cfg)

	if err != nil {
		log.Warn("failed to decode configuration:", err)
		return
	}
}

func saveCfg() {
	data, err := json.MarshalIndent(Cfg, "", "\t")
	if err != nil {
		log.Fatal(err)
	}

	err = os.WriteFile(path()+"/config.json", data, 0o666)
	if err != nil {
		log.Err(err)
	}
}

func path() string {
	return "."
	/*ex, err := os.Executable()
	if err != nil {
		panic(err)
	}
	exPath := filepath.Dir(ex)
	return strings.TrimSuffix(exPath, string(os.PathSeparator))*/
}
