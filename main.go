package main

import (
	"flag"
	"os"
	"runtime"
	"strings"
	"xelis-mining-proxy/log"

	"github.com/TwiN/go-color"
)

func main() {
	walletAddr := ""
	url := ""
	protocol := ""
	debug := false
	save := false

	flag.StringVar(&walletAddr, "wallet", "", "your xelis address")
	flag.StringVar(&url, "url", "", "mining pool url")
	flag.StringVar(&protocol, "protocol", "", "mining pool protocol, possible values: xatum, getwork")
	flag.BoolVar(&debug, "debug", false, "true if you want to make logs verbose")
	flag.BoolVar(&save, "save-config", false, "force saving the config to a json file")
	flag.Parse()

	if debug || Cfg.Debug {
		log.Info("debug mode ON")
		Cfg.Debug = true
		log.LogLevel = 2
	}
	if walletAddr != "" && walletAddr != "YOUR_WALLET_ADDRESS" {
		Cfg.WalletAddress = walletAddr
	}
	if url != "" {
		Cfg.PoolUrl = url
	}
	if protocol != "" {
		Cfg.PoolProtocol = protocol
	}
	if save {
		saveCfg()
	}

	if Cfg.WalletAddress == "YOUR_WALLET_ADDRESS" {
		Cfg.WalletAddress = StringPrompt("Enter your wallet address:")

		if len(Cfg.WalletAddress) > 10 {
			saveCfg()
		} else {
			log.Err("invalid wallet address")
			os.Exit(0)
		}
	}

	log.Title("")
	log.Title(color.InBold("XELIS-MINING-PROXY v" + VERSION))
	log.Title(color.Ize(color.Purple, "https://github.com/xelis-project/xelis-mining-proxy"))
	log.Title("")
	log.Title(color.Cyan+"OS:", runtime.GOOS, "arch:", runtime.GOARCH, "threads:", runtime.NumCPU())
	log.Title(color.Reset + "")

	Cfg.PoolProtocol = strings.ToLower(Cfg.PoolProtocol)

	switch Cfg.PoolProtocol {
	case "stratum", "getwork":
		log.Info("Using pool protocol:", Cfg.PoolProtocol)
	case "auto":
		Cfg.PoolProtocol = "stratum"
		splUrl := strings.Split(Cfg.PoolUrl, ":")
		if len(splUrl) > 2 {
			splUrl = splUrl[1:]
		}

		if len(splUrl) > 1 {
			port := splUrl[1]

			if port == "8080" || port == "2086" {
				Cfg.PoolProtocol = "getwork"
			}
		}
		log.Info("Automatically selected protocol:", Cfg.PoolProtocol)
	}

	go listenGetwork()
	go listenStratum(stratumServer)

	if Cfg.PoolProtocol == "getwork" {
		getworkClientHandler()
	} else {
		panic("stratum pools not implemented yet")
	}
}
