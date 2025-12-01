package main

import (
	"encoding/hex"
	"strconv"
	"strings"
	"time"
	"xelis-mining-proxy/log"
	"xelis-mining-proxy/stratum"
	"xelis-mining-proxy/util"

	"github.com/xelis-project/xelis-go-sdk/getwork"
)

// Getwork client

// Share represents a share to be sent to the pool
type Share struct {
	ID      string // Unique identifier: hex(extra_nonce + nonce)
	Encoded string // minerWork hex encoded string
}

var clGw *getwork.Getwork
var sharesToPool chan Share
var shareTracker *ShareTracker
var pendingShareQueue chan string // FIFO queue of share IDs in submission order

func getworkClientHandler() {
	func() {
		prefix := strings.Split(Cfg.PoolUrl, ":")[0]
		if prefix != "ws" && prefix != "wss" {
			Cfg.PoolUrl = "ws://" + Cfg.PoolUrl
		}
	}()

	log.Debug("getwork pool url", Cfg.PoolUrl)

	// Initialize share tracker with 30 second timeout
	shareTracker = NewShareTracker(30 * time.Second)

	for {
		log.Info("Starting a new connection to the pool")

		sharesToPool = make(chan Share, 1)
		pendingShareQueue = make(chan string, 100) // Buffer for pending shares

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

		log.Debugf("Share ID: %s, Encoded: %s", share.ID, share.Encoded)

		// Add to FIFO queue BEFORE submitting to pool
		pendingShareQueue <- share.ID

		err := clGw.SubmitBlock(share.Encoded)
		if err != nil {
			log.Err("failed to submit share to pool:", err)

			// On submit error, remove from queue and send rejection
			<-pendingShareQueue // Remove from queue
			if pending := shareTracker.GetPendingShare(share.ID); pending != nil {
				pending.ResponseChan <- ShareResult{
					Accepted: false,
					Error: &stratum.Error{
						Code:    -1,
						Message: "failed to submit to pool",
					},
				}
			}

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
		go stratumServer.sendJobs()
	}
}

func readAcceptGw() {
	for {
		accepted, ok := <-clGw.AcceptedBlock
		if !ok {
			return
		}

		log.Info("share accepted:", accepted)

		// Get the next share ID from FIFO queue
		shareID, ok := <-pendingShareQueue
		if !ok {
			log.Warn("pendingShareQueue closed")
			return
		}

		// Send acceptance to the waiting miner
		if pending := shareTracker.GetPendingShare(shareID); pending != nil {
			log.Debugf("Matched accepted share %s to pending share", shareID)
			pending.ResponseChan <- ShareResult{
				Accepted: true,
				Error:    nil,
			}
		} else {
			log.Warnf("Received accept for unknown or expired share: %s", shareID)
		}
	}
}

func readRejectGw() {
	for {
		rejectReason, ok := <-clGw.RejectedBlock
		if !ok {
			return
		}

		log.Err("share rejected:", rejectReason)

		// Get the next share ID from FIFO queue
		shareID, ok := <-pendingShareQueue
		if !ok {
			log.Warn("pendingShareQueue closed")
			return
		}

		// Send rejection to the waiting miner
		if pending := shareTracker.GetPendingShare(shareID); pending != nil {
			log.Debugf("Matched rejected share %s to pending share", shareID)
			pending.ResponseChan <- ShareResult{
				Accepted: false,
				Error: &stratum.Error{
					Code:    -1,
					Message: "rejected by pool: " + rejectReason,
				},
			}
		} else {
			log.Warnf("Received reject for unknown or expired share: %s", shareID)
		}
	}
}
