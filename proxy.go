package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
	"xelis-mining-proxy/xelisutil"
)

const VERSION = "1.0.0"

// Job is a fast & efficient struct used for storing a job in memory
type Job struct {
	Blob   xelisutil.BlockMiner
	Diff   uint64
	Target [32]byte
}

var stratumServer = &StratumServer{
	Conns: make([]*StratumConn, 0),
}

func StringPrompt(label string) string {
	var s string
	r := bufio.NewReader(os.Stdin)
	for {
		fmt.Fprint(os.Stderr, label+" ")
		s, _ = r.ReadString('\n')
		if s != "" {
			break
		}
	}
	return strings.TrimSpace(s)
}

var curJob Job
var mutCurJob sync.RWMutex
