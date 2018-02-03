package main

import (
	"fmt"
	"net/rpc"
	"sync"
)

// Static
var canvasWidth int
var canvasHeight int
var minConn int

// Current block
var currBlock *Block
var blockLock = &sync.Mutex{}

// Network
var blockTree map[string]*Block
var headBlock *Block
var serverConn *rpc.Client
var neighbours []*InkMiner

// FIXME
var ink int // TODO Do we want this? Or do we want a func that scans blockchain before & after op validation

type Point struct {
	x, y int
}

type Shape struct {
	hash     string
	svg      string
	point    []Point
	filledIn bool
	ink      int
}

type Op struct {
	shape     *Shape // not nil iff adding shape
	shapeHash string // non-empty iff removing shape
	owner     string // hash of pub/priv keys
}

type Block struct {
	prev  string
	ops   []Op
	nonce string
}

type InkMiner struct {
	conn *rpc.Client
}

// TODO: Validate stub
// Sends op to all neighbours
// @param op: Op to be broadcast
func floodOps(op Op) {
	// TODO -- should prob be async. See rpc.Go & select
}

// TODO: Validate stub
// Sends block to all neighbours
// @param block: Block to be broadcast
func floodBlock(block Block) {
	// TODO -- should prob be async. See rpc.Go & select
}

// TODO: Validate stub
// Continually searches for nonce for the global currBlock.
// Runs on seperate thread. All interactions should take place
// over a chan, or through a Mutex.
func solveNonce() {
	for {
		// TODO
	}
}

// TODO: Validate stub
// - Validates an operation
// - Adds it to currBlock's ops list
// - Floods ops to neighbours
func validateOps(op Op) {
	// TODO
}

func main() {
	fmt.Println("vim-go")
}
