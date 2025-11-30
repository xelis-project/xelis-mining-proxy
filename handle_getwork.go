package main

import (
	"encoding/hex"
	"strconv"
	"strings"
	"time"
	"xelis-mining-proxy/log"
	"xelis-mining-proxy/util"

	"github.com/xelis-project/xelis-go-sdk/getwork"
)

// Getwork client

// Share represents a share to be sent to the pool
// it is a minerWork hex encoded string
type Share string

var clGw *getwork.Getwork
var sharesToPool chan Share

func getworkClientHandler() {
	func() {
		prefix := strings.Split(Cfg.PoolUrl, ":")[0]
		if prefix != "ws" && prefix != "wss" {
			Cfg.PoolUrl = "ws://" + Cfg.PoolUrl
		}
	}()

	log.Debug("getwork pool url", Cfg.PoolUrl)

	for {
		log.Info("Starting a new connection to the pool")

		sharesToPool = make(chan Share, 1)

		var err error
		clGw, err = getwork.NewGetwork(Cfg.PoolUrl+"/getwork", Cfg.WalletAddress, "xelis-mining-proxy v"+VERSION)
		if err != nil {
			log.Err(err)
			time.Sleep(time.Second)
			continue
		}

		go recvSharesGw(clGw)
		go readAcceptGw()
		go readRejectGw()

		readjobsGw(clGw)

		// close(sharesToPool)

		log.Debug("pool connection closed, starting a new one")

		time.Sleep(time.Second)
	}
}
func recvSharesGw(clGw *getwork.Getwork) {
	log.Debug("recvShares started")
	for {
		share, ok := <-sharesToPool
		if !ok {
			log.Warn("sharesToPool chan closed")
			clGw.Close()
			return
		}

		log.Info("Share found, submitting to pool")

		log.Debugf("%x", share)

		err := clGw.SubmitBlock(string(share))
		if err != nil {
			log.Err("failed to submit share to pool:", err)
			clGw.Close()
			return
		}
	}
}
func readjobsGw(clGw *getwork.Getwork) {
	for {
		job, ok := <-clGw.Job
		if !ok {
			return
		}

		tmpl, err := hex.DecodeString(job.Template)
		if err != nil {
			log.Err(err)
			clGw.Close()
			return
		}

		diff, err := strconv.ParseUint(job.Difficulty, 10, 64)
		if err != nil {
			log.Err(err)
			clGw.Close()
			return
		}

		if len(tmpl) != util.BLOCKMINER_LENGTH {
			log.Errf("template %x length is invalid", tmpl)
			clGw.Close()
			return
		}

		log.Debug("new job from GetWork")

		bm := util.BlockMiner(tmpl)

		mutCurJob.Lock()
		curJob = Job{
			Blob:       bm,
			Diff:       diff,
			Target:     util.GetTargetBytes(diff),
			Algorithm:  job.Algorithm,
			Height:     job.Height,
			TopoHeight: job.TopoHeight,
		}
		mutCurJob.Unlock()

		log.Infof("new job with difficulty %d for algorithm %s", diff, job.Algorithm)
		log.Debugf("new job: diff %d, blob %x", diff, tmpl)

		log.Debugf("blob public key %x", util.BlockMiner(tmpl).GetPublickey())

		go sendJobToWebsocket(diff, tmpl)
		go stratumServer.sendJobs(diff, bm)
	}
}

func readAcceptGw() {
	for {
		accept, ok := <-clGw.AcceptedBlock
		if !ok {
			return
		}

		log.Info("share accepted:", accept)
	}
}

func readRejectGw() {
	for {
		reject, ok := <-clGw.RejectedBlock
		if !ok {
			return
		}

		log.Err("share rejected:", reject)
	}
}
