package main

import (
	"encoding/hex"
	"strconv"
	"strings"
	"time"
	"xelis-mining-proxy/log"
	"xelis-mining-proxy/xelisutil"

	"github.com/xelis-project/xelis-go-sdk/getwork"
)

// Getwork client

var clGw *getwork.Getwork
var sharesToPool chan xelisutil.PacketC2S_Submit

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

		sharesToPool = make(chan xelisutil.PacketC2S_Submit, 1)

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

		bm := share.BlockMiner

		log.Debugf("%x", bm)
		log.Debug(bm.String())

		err := clGw.SubmitBlock(share.BlockMiner.String())
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

		if len(tmpl) != xelisutil.BLOCKMINER_LENGTH {
			log.Errf("template %x length is invalid", tmpl)
			clGw.Close()
			return
		}

		log.Debug("new job from GetWork")

		bm := xelisutil.BlockMiner(tmpl)

		mutCurJob.Lock()
		curJob = Job{
			Blob:   bm,
			Diff:   diff,
			Target: xelisutil.GetTargetBytes(diff),
		}
		mutCurJob.Unlock()

		log.Infof("new job with difficulty %d", diff)
		log.Debugf("new job: diff %d, blob %x", diff, tmpl)

		log.Debugf("blob public key %x", xelisutil.BlockMiner(tmpl).GetPublickey())

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
