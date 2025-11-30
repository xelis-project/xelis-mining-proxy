package main

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
	"xelis-mining-proxy/config"
	"xelis-mining-proxy/log"
	"xelis-mining-proxy/stratum"
	"xelis-mining-proxy/util"
)

// Stratum server

const JOBS_PAST = 5

type PastJob struct {
	JobID      [16]byte
	BlockMiner util.BlockMiner
}

type StratumServer struct {
	Conns []*StratumConn

	sync.RWMutex
}

type StratumConn struct {
	Conn      net.Conn
	Alive     bool
	IP        string
	LastOutID uint32
	Jobs      []PastJob
	Agent     string
	Ready     bool

	sync.RWMutex
}

// StratumConn MUST be locked before calling this
func (g *StratumConn) WriteJSON(data any) error {
	bin, err := json.Marshal(data)

	if err != nil {
		return err
	}

	log.Debug("stratum >>>", string(bin))

	_, err = g.Conn.Write(append(bin, []byte("\n")...))
	return err
}

func (g *StratumConn) Close() error {
	g.Alive = false
	return g.Conn.Close()
}

func listenStratum(s *StratumServer) {

	if Cfg.StratumBindPort == 0 {
		Cfg.StratumBindPort = 5209
		go saveCfg()
	}

	listener, err := net.Listen("tcp", "0.0.0.0:"+strconv.FormatUint(uint64(Cfg.StratumBindPort), 10))
	if err != nil {
		log.Fatal(err)
	}

	log.Infof("Stratum server listening on port %d", Cfg.StratumBindPort)

	// Start the pinger
	go func() {
		for {
			time.Sleep((config.SLAVE_MINER_TIMEOUT - 5) * time.Second)

			s.Lock()
			for _, v := range s.Conns {
				go func() {
					v.Lock()
					defer v.Unlock()

					v.LastOutID++
					v.WriteJSON(stratum.RequestOut{
						Id:     v.LastOutID,
						Method: "mining.ping",
						Params: nil,
					})
				}()
			}
			s.Unlock()
		}
	}()

	// Accept incoming connections and handle them
	for {
		Conn, err := listener.Accept()
		if err != nil {
			fmt.Println(err)
			continue
		}

		ip := util.RemovePort(Conn.RemoteAddr().String())

		sConn := &StratumConn{
			Conn: Conn,
			Jobs: make([]PastJob, 0, JOBS_PAST),
		}

		sConn.Alive = true
		sConn.IP = ip

		s.Lock()
		s.Conns = append(s.Conns, sConn)
		s.Unlock()

		// Handle the connection in a new goroutine
		go handleStratumConn(s, sConn)
	}
}

func GenerateID() [16]byte {
	id := make([]byte, 16)
	rand.Read(id)
	return [16]byte(id)
}

func handleStratumConn(_ *StratumServer, c *StratumConn) {
	rdr := bufio.NewReader(c.Conn)

	numMessages := 0

	for {
		c.Lock()
		var err error
		if numMessages < 2 {
			err = c.Conn.SetReadDeadline(time.Now().Add(config.TIMEOUT * time.Second))
		} else {
			err = c.Conn.SetReadDeadline(time.Now().Add(config.SLAVE_MINER_TIMEOUT * time.Second))
		}
		c.Unlock()
		numMessages++

		if err != nil {
			log.Warn("Miner", c.IP, "disconnected:", err)
			c.Conn.Close()
			c.Alive = false
			return
		}

		str, err := rdr.ReadString('\n')
		if err != nil {
			log.Warn("Miner", c.IP, "disconnected:", err)
			c.Conn.Close()
			c.Alive = false
			return
		}

		log.Debug("stratum <<<", str)

		req := stratum.RequestIn{}

		err = json.Unmarshal([]byte(str), &req)
		if err != nil {
			log.Warn(err)
			c.Close()
			c.Alive = false
			return
		}

		switch req.Method {
		case "mining.subscribe":
			params := []any{}
			err := json.Unmarshal(req.Params, &params)
			if err != nil {
				log.Warn(err)
				c.Close()
				c.Alive = false
				return
			}

			c.Agent = params[0].(string)

			log.Info("Stratum miner with agent", c.Agent, "and IP", c.IP, "connected")

			mutCurJob.RLock()
			job := curJob
			mutCurJob.RUnlock()

			if len(params) < 1 {
				log.Warn("less than 1 param")
				c.Close()
				c.Alive = false
				return
			}

			// generate a random extra nonce for the miner
			job.Blob.GenerateExtraNonce()
			c.Lock()

			log.Debugf("sending Stratum informations to miner with IP %s", c.IP)

			xnonce := job.Blob.GetExtraNonce()
			pubkey := job.Blob.GetPublickey()

			if pubkey == [32]byte{} {
				c.WriteJSON(stratum.ResponseOut{
					Id: req.Id,
					Error: &stratum.Error{
						Code:    -1,
						Message: "no job yet",
					},
				})
				c.Close()
				return
			}

			err = c.WriteJSON(stratum.ResponseOut{
				Id: req.Id,
				Result: []any{
					"",                            // useless (session id)
					hex.EncodeToString(xnonce[:]), // extra nonce
					32,                            // useless (extra nonce length)
					hex.EncodeToString(pubkey[:]), // public key
				},
			})

			if err != nil {
				log.Warn(err)
				c.Alive = false
				c.Unlock()
				return
			}
			c.Unlock()
		case "mining.authorize":
			params := []string{}

			err := json.Unmarshal(req.Params, &params)
			if err != nil {
				log.Warn(err)
				c.Close()
				c.Alive = false
				return
			}

			if len(params) < 3 {
				log.Warn("less than 3 params")
				c.Close()
				c.Alive = false
				return
			}

			params[0] = strings.ReplaceAll(params[0], ".", "+")

			splAddr := strings.Split(params[0], "+")

			wall := splAddr[0]

			log.Info("Stratum miner with address", wall, "IP", c.IP, "connected")
			c.Alive = true

			// send the job
			mutCurJob.RLock()
			job := curJob
			mutCurJob.RUnlock()

			// first, send response
			c.Lock()
			err = c.WriteJSON(stratum.ResponseOut{
				Id:     req.Id,
				Result: true,
			})
			if err != nil {
				log.Warn("failed to send response")
				c.Close()
				c.Unlock()
				return
			}

			// send actual job
			SendStratumJob(c, job)

			c.Unlock()

		case "mining.submit":
			params := []string{}

			err := json.Unmarshal(req.Params, &params)
			if err != nil {
				log.Warn(err)
				c.Close()
				c.Alive = false
				return
			}

			if len(params) != 3 {
				log.Warn("params length is not 3")
				c.Close()
				c.Alive = false
				return
			}

			jid, err := hex.DecodeString(params[1])
			if err != nil {
				log.Warn(err)
				c.Close()
				c.Alive = false
				return
			}
			nonceBin, err := hex.DecodeString(params[2])

			if err != nil {
				log.Warn(err)
				c.Close()
				c.Alive = false
				return
			}

			if len(jid) != 16 || len(nonceBin) != 8 {
				log.Warnf("jobid %x nonce %x do not match expected length (16, 8)", jid, nonceBin)
				c.Close()
				c.Alive = false
				return
			}

			jobid := [16]byte(jid)

			// get the BlockMiner for the current job
			var bm util.BlockMiner
			found := false
			for _, v := range c.Jobs {
				if v.JobID == jobid {
					log.Debugf("job id %x matches", jobid)
					bm = v.BlockMiner
					log.Debugf("blockMiner is %x", bm)
					found = true

					// the bug is before this
					break
				}
				log.Debugf("job id %x doesn't match with %x", jobid, v.JobID)
			}

			if !found {
				log.Warnf("unknown job id %x, share is probably stale", jobid)

				c.WriteJSON(stratum.ResponseOut{
					Id:     req.Id,
					Result: false,
					Error: &stratum.Error{
						Code:    -1,
						Message: "stale share",
					},
				})

				return
			}

			bm.SetNonceBytes([8]byte(nonceBin))

			go func() {
				c.Lock()
				defer c.Unlock()

				c.WriteJSON(stratum.ResponseOut{
					Id:     req.Id,
					Result: true,
				})
			}()

			encoded := bm.String()

			log.Info("Stratum miner with IP", c.IP, "found a share for job id", jobid, "nonce", bm.GetNonce())

			log.Debugf("share blob %x", encoded)

			// send share to pool
			sharesToPool <- Share(encoded)
		default:
			if req.Method != "mining.pong" {
				log.Warn("Unknown Stratum method", req.Method)
			}
		}

	}

}

// NOTE: StratumConn MUST be locked before calling this
func (c *StratumConn) SendDifficulty(diff uint64) error {
	c.LastOutID++

	if diff == 0 {
		diff = 18446744073709551615
	}

	return c.WriteJSON(stratum.RequestOut{
		Id:     c.LastOutID,
		Method: "mining.set_difficulty",
		Params: []uint64{diff},
	})
}

func GenerateJobID() [16]byte {
	b := make([]byte, 16)

	rand.Read(b)

	return [16]byte(b)
}

func (c *StratumConn) SendJob(bm util.BlockMiner, jobid [16]byte, job Job) error {
	c.LastOutID++

	workhash := bm.GetWorkhash()

	timeStr := strconv.FormatUint(bm.GetTimestamp(), 16)

	xn := bm.GetExtraNonce()

	err := c.WriteJSON(stratum.RequestOut{
		Id:     c.LastOutID,
		Method: "mining.set_extranonce",
		Params: []any{
			hex.EncodeToString(xn[:]),
			32,
		},
	})

	if err != nil {
		return err
	}

	c.LastOutID++

	algorithm := strings.ReplaceAll(job.Algorithm, "v", "")

	return c.WriteJSON(stratum.RequestOut{
		Id:     c.LastOutID,
		Method: "mining.notify",
		Params: []any{
			hex.EncodeToString(jobid[:]),
			timeStr,
			hex.EncodeToString(workhash[:]),
			algorithm,
			true,
		},
	})
}

func SendStratumJob(v *StratumConn, job Job) {
	log.Debug("SendJob to Stratum miner with IP", v.Conn.RemoteAddr().String())

	jobId := make([]byte, 16)
	_, err := rand.Read(jobId)
	if err != nil {
		log.Err(err)
		return
	}

	// generate a random extra nonce for the miner
	blob := job.Blob
	blob.GenerateExtraNonce()

	log.Debugf("SendStratumJob blob %x", blob)
	log.Debug("SendStratumJob:", blob.Display())

	// add the job to miner's known past jobs
	v.Jobs = append(v.Jobs, PastJob{
		JobID:      [16]byte(jobId),
		BlockMiner: blob,
	})
	if len(v.Jobs) > JOBS_PAST {
		v.Jobs = v.Jobs[1:]
	}

	log.Debugf("sending job to Stratum miner with IP %s (job id %x) ok", v.IP, jobId)

	v.SendDifficulty(job.Diff)
	v.SendJob(blob, [16]byte(jobId), job)
}

// sends a job to all the websockets, and removes old websockets
func (s *StratumServer) sendJobs() {
	s.Lock()
	log.Debug("StratumServer sendJobs: num sockets:", len(s.Conns))

	// remove disconnected sockets

	sockets2 := make([]*StratumConn, 0, len(s.Conns))
	for _, c := range s.Conns {
		if c == nil {
			log.Err("THIS SHOULD NOT HAPPEN - connection is nil")
			continue
		}
		if !c.Alive {
			log.Debug("connection with IP", c.IP, "disconnected")

			continue
		}
		sockets2 = append(sockets2, c)
	}
	log.Debug("StratumServer sendJobs: going from", len(s.Conns), "to", len(sockets2), "Stratum miners")
	s.Conns = sockets2

	if len(s.Conns) > 0 {
		log.Info("Sending job to", len(s.Conns), "Stratum miners")
	}
	s.Unlock()

	// send jobs to the remaining sockets

	for _, cx := range sockets2 {
		if cx == nil {
			log.Debug("cx is nil")
			continue
		}

		c := cx

		// send job in a new thread to avoid blocking the main thread and reduce latency
		go func() {
			log.Debug("StratumServer sendJobs: sending to IP", c.IP)

			c.Lock()
			defer c.Unlock()

			SendStratumJob(c, curJob)

			log.Debug("StratumServer sendJobs: done, sent to IP", c.IP)
		}()
	}
}
