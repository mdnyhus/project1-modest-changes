package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"net/rpc"
	"strings"
	"sync"
	"time"
	"./blockartlib"
)

// Static
var canvasWidth int
var canvasHeight int
var minConn int
var n int // Num 0's required in POW

// Current block
var currBlock *Block
var blockLock = &sync.Mutex{}

// Head block
var headBlock *Block
var headBlockLock = &sync.Mutex{}

// Neighbours
var neighbours []*InkMiner
var neighboursLock = &sync.Mutex{}

// Network
var blockTree map[string]*Block
var serverConn *rpc.Client

// FIXME
var ink int // TODO Do we want this? Or do we want a func that scans blockchain before & after op validation

type Op struct {
	shape     *blockartlib.Shape // not nil iff adding shape
	shapeHash string // non-empty iff removing shape
	owner     string // hash of pub/priv keys
}

type Block struct {
	prev  string
	ops   []Op
	len   int
	nonce string
}

func (b Block) String() string {
	return fmt.Sprintf("%+v", b)
}

type InkMiner struct {
	conn *rpc.Client
}

// RPC type responsible for Miner-to-Miner communcation.
type MinMin int

type BlockNotFoundError string

func (e BlockNotFoundError) Error() string {
	return fmt.Sprintf("InkMiner: Could not find block with hash %s", string(e))
}

type BlockVerificationError string

func (e BlockVerificationError) Error() string {
	return fmt.Sprintf("InkMiner: Block with hash %s could not be verified", string(e))
}

// Receives op block flood calls. Verifies the op, which will add the op to currBlock and flood
// op if it is valid.
// @param op *Op: Op which will be verified, and potentially added and flooeded
// @param reply *bool: Bool indicating whether op was successfully validated
// @return error: TODO
func (m *MinMin) NotifyNewOp(op *Op, reply *bool) (err error) {
	// if op is validated, validateOp will put op in currBlock itself and flood the op
	*reply = validateOp(*op)
	return nil
}

// TODO RPC calls feel a bit burdensome here.
// Receives block flood calls. Verifies chains. Updates head block if new chain is acknowledged.
// LOCKS: Calls headBlockLock()
// @param block *Block: Block which was added to chain.
// @param reply *bool: Bool indicating success of RPC.
// @return error: Any errors produced during new block processing.
func (m *MinMin) NotifyNewBlock(block *Block, reply *bool) error {
	*reply = false
	len := 0
	currBlock := block

	for !isGenesis(*currBlock) {
		if verifyHash(hashBlock(*currBlock)) {
			len++
			currBlock = getBlock(currBlock.prev)
			if currBlock == nil {
				// Could not verify due to missing block in chain.
				return BlockVerificationError(block.nonce)
			}
		} else {
			// Could not verify due to invalid nonce in chain.
			return BlockVerificationError(block.nonce)
		}
	}

	if len != block.len {
		// Could not verify due to block len claim inconsistencies.
		return BlockVerificationError(block.nonce)
	}

	*reply = true
	headBlockLock.Lock()
	defer headBlockLock.Unlock()

	if len > headBlock.len {
		headBlock = block
	}

	return nil
}

// Returns block identified with provided nonce.
// @param nonce *string: Nonce of block to be returned.
// @param block *Block: Pointer to block specified by nonce.
// @return error: Any errors produced in retrieval of block.
func (m *MinMin) RequestBlock(hash *string, block *Block) error {
	block = blockTree[*hash]
	if block == nil {
		return BlockNotFoundError(*hash)
	}
	return nil
}

// Returns block with given nonce. Will search neighbours if not found locally.
// @param nonce string: The nonce of the block to get info on.
// @return Block: The requested block, or nil if no block is found.
func getBlock(hash string) (block *Block) {
	// Search locally.
	if block = blockTree[hash]; block != nil {
		return block
	}

	for _, n := range neighbours {
		err := n.conn.Call("MinMin.RequestBlock", hash, block)
		if err != nil {
			// Block not found, keep searching.
			continue
		}
		// Save block locally.
		blockTree[hash] = block
		return block
	}

	// Block not found.
	return nil
}

// Returns true if block is the genesis block.
// @param block Block: The block to test against.
// @return bool: True iff block is genesis block.
func isGenesis(block Block) bool {
	// TODO: def'n of Genesis block?
	return block.prev == ""
}

// Returns hash of block.
// @param block Block: Block to be hashed.
// @return string: The hash of the block.
func hashBlock(block Block) string {
	hasher := md5.New()
	hasher.Write([]byte(block.String()))
	return hex.EncodeToString(hasher.Sum(nil))
}

// Verifies that hash meets POW requirements specified by server.
// @param hash string: Hash of block to be verified.
func verifyHash(hash string) bool {
	return hash[len(hash)-n:] == strings.Repeat("0", n)
}

// TODO this and floodBlock currentl share almost all the code. If worth it, call helper
//      function that takes the function and paramters.
// Sends op to all neighbours.
// LOCKS: Calls neighboursLock.Lock().
// @param op Op: Op to be broadcast.
func floodOp(op Op) {
	// Prevent other processes from adding/removing neighbours.
	neighboursLock.Lock()
	defer neighboursLock.Unlock()

	replies := 0
	replyChan := make(chan *rpc.Call, 1)

	for _, n := range neighbours {
		var reply bool
		_ = n.conn.Go("NotifyNewOp", op, &reply, replyChan)
	}

	// TODO: Handle errors, chain disagreements. Discuss with team.
	// Current implementation simply sends out blocks and doesn't
	// care about the response.
	for replies != len(neighbours) {
		select {
		case <-replyChan:
			replies++
		case <-time.After(2 * time.Second):
			// TODO Do we care? Noop for now.
			replies++
		}
	}
}

// Sends block to all neighbours.
// @param block Block: Block to be broadcast.
func floodBlock(block Block) {
	// Prevent other processes from adding/removing neighbours.
	neighboursLock.Lock()
	defer neighboursLock.Unlock()

	replies := 0
	replyChan := make(chan *rpc.Call, 1)

	for _, n := range neighbours {
		var reply bool
		_ = n.conn.Go("NotifyNewBlock", block, &reply, replyChan)
	}

	// TODO: Handle errors, chain disagreements. Discuss with team.
	// Current implementation simply sends out blocks and doesn't
	// care about the response.
	for replies != len(neighbours) {
		select {
		case <-replyChan:
			replies++
		case <-time.After(2 * time.Second):
			// TODO Do we care? Noop for now.
			replies++
		}
	}
}

// TODO: Validate stub.
// Continually searches for nonce for the global currBlock.
// Runs on seperate thread. All interactions should take place
// over a chan, or through a Mutex.
func solveNonce() {
	for {
		// TODO
	}
}

// TODO: Validate stub.
// - Validates an operation.
// - If validated, adds it to currBlock's ops list, floods ops to neighbours, and returns true.
// - if not validated, returns false and ignores the op.
// LOCKS: Calls blockLock.Lock()
// @param op Op: Op to be validated.
// @return bool: True if op is valid, false otherwise
func validateOp(op Op) bool {
	if op.shape != nil {
		if !validateShape(shape *blockartlib.Shape) {
			// shape is not valid
			return false
		}
	}

	blockLock.Lock()
	defer blockLock.Unlock()

	if isMyOp(op) && op.shape != nil {
		// check if miner has enough ink only if op is owned by this miner
		// and if shape is not being deleted
		ink, err := blockartlib.InkUsed(op.shape)
		if err != nil || countInk() < ink {
			// not enough ink
			return false
		}
	}

	if op.shape != nil && shapeIntersects(op.shape) {
		// op is adding a shape that intersects with an already present shape; reject
		return false
	}

	if op.shape == nil && !shapeExists(op.shapeHash) {
		// Op is trying to delete a shape that has been deleted
		return false
	}

	// op is valid; add op to currBlock
	currBlock.ops = append(currBlock.ops, op)
	// floodOp on a separate thread; this miner's operation doesn't depend on the flood
	go floodOp(op)
	return true
}

// TODO
// - checks op's hash and miner's public/private key to decide if 
//   op belongs to this miner
// @param op Op: Op to be checked
// @return isMine bool: true if op belongs to this miner, false otherwise
func isMyOp(op Op) (isMyOp bool) {
	// should use op.owner
	return false
}

// TODO
// Checks if the passed shape is valid according to the spec
// - TODO shape fill spec re. convex or self-intersections
// - shape points are within the canvas
// @param shape *blockartlib.Shape: pointer to shape that will be validated
// @return valid bool: true if shape is valid, false otherwise 
func validateShape(shape *blockartlib.Shape) (valid bool) {
	// TODO
	return false
}

// TODO
// - checks if the passed shape intersects with any shape currently on the canvas
//   that is NOT owned by this miner
// - ASSUMES that the blockLock has already been aquired
// @param shape *blockartlib.Shape: pointer to shape that will be checked for 
//                                  intersections
// @return shapeIntersects bool: true if shape does intersect with a shape 
//                               currently on the canvas, false otherwise
func shapeIntersects(shape *blockartlib.Shape) (shapeIntersects bool) {
	// TODO
	return false
}

// TODO
// - checks if a shape with the given hash exists on the canvas (and was not 
//   later deleted)
// - ASSUMES that the blockLock has already been aquired
// @param shapeHash string: hash of shape to check
// @return shapeExists bool: true if shape does exist on the canvas, 
//                           false otherwise
func shapeExists(shapeHash string) (shapeExists bool) {
	// TODO
	return false
}

// TODO
// - counts the amount of ink currently available
// - ASSUMES that the blockLock has already been aquired
// @return ink int: ink currently available to this miner, in pixels
func countInk() (ink int) {
	// TODO
	// Depends on starting ink, and how much ink you receive for each new block
	return 0
}

func main() {
	fmt.Println("vim-go")
}
