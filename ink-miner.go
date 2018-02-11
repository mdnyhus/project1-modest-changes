/*
Implements the ink-miner for project 1 for UBC CS 416 2017 W2.

Usage:
$ go run ink-miner.go [client-incoming ip:port]

Example:
$ go run ink-miner.go 127.0.0.1:2020

*/

package main

import (
	"crypto/md5"
	"encoding/hex"
	"fmt"
	"os"
	"net"
	"net/rpc"
	"strings"
	"sync"
	"time"
	"./blockartlib"
)

// Static
var canvasSettings balib.CanvasSettings
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
var address string

// FIXME
var ink int // TODO Do we want this? Or do we want a func that scans blockchain before & after op validation

type Op struct {
	shape     *balib.Shape // not nil iff adding shape
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
	// TODO - check if op has already been seen, and only flood if it is new
	// if op is validated, receiveNewOp will put op in currBlock and flood the op
	*reply = false
	if e := receiveNewOp(*op); e == nil {
		// validate was successful only if error is null
		// TODO - is the error  useful?
		*reply = true
	}
	return nil
}

// Receives block flood calls. Verifies chains. Updates head block if new chain is acknowledged.
// LOCKS: Acquires and releases headBlockLock
// @param block *Block: Block which was added to chain.
// @param reply *bool: Bool indicating success of RPC.
// @return error: Any errors produced during new block processing.
func (m *MinMin) NotifyNewBlock(block *Block, reply *bool) error {
	if b := blockTree[hashBlock(*block)]; b != nil {
		// We are already aware of this block.
		return nil
	}

	*reply = false
	len := 0
	curr := block

	for !isGenesis(*curr) {
		// TODO: Verify ops.
		if verifyBlock(*curr) {
			len++
			currBlock = getBlock(curr.prev)
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

	floodBlock(block)

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

// RPC for blockartlib-miner connection
type LibMin int

// Returns the CanvasSettings
// @param args int: required by Go's RPC; does nothing
// @param reply *balib.ConvasSettings: pointer to CanvasSettings that will be returned
// @return error: Any errors produced
func (l *LibMin) GetCanvasSettings(args int, reply *balib.CanvasSettings) (err error) {
	*reply = canvasSettings
	return nil
}

// Adds a new shape ot the canvas
// @param args *balib.AddShapeArgs: contains the shape to be added, and the validateNum
// @param reply *balib.AddShapeReply: pointer to AddShapeReply that will be returned
// @return error: Any errors produced
func (l *LibMin) AddShape(args *balib.AddShapeArgs, reply *balib.AddShapeReply) (err error) {
	// construct Op for shape
	op := Op{
		shape: &args.Shape,
		shapeHash: "",
		owner: "", // TODO - generate owner hash
	}
	
	// receiveNewOp will try to add op to current block and flood op
	if err = receiveNewOp(op); err != nil {
		// return error in reply so that it is not cast
		reply.Error = err
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

// Returns true if block is valid.
// @param block Block: The block to be verified.
// @return bool: Whether the block is valid.
func verifyBlock(block Block) bool {
	return verifyHash(hashBlock(block)) && verifyOps(block)
}


// Verifies that hash meets POW requirements specified by server.
// @param hash string: Hash of block to be verified.
// @return bool: True iff valid.
func verifyHash(hash string) bool {
	return hash[len(hash)-n:] == strings.Repeat("0", n)
}

// Verifies that all ops are valid and no shape conflicts exist against blockchain canvas.
// @param ops []Op: Slice of ops to verify.
// @return bool: True iff valid.
func verifyOps(block Block) bool {
	for i, op := range block.ops {
		if !verifyShape(op.shape) {
			return false
		}

		// Ensure op does not conflict with previous ops.
		for j := 0; j < i; j++ {
			if jOp := block.ops[j]; op.owner != jOp.owner {
				if balib.ShapesIntersect(op.Shape, jOp.Shape) {
					return false
				}
			}
		}
	}
}

// Verifies Shape corresponds to SVG string.
// @param shape Shape: The shape to verify.
// @return bool: True iff valid.
func verifyShape(shape Shape) bool {
	parsedSvg, err := balib.ParseSvgPath(shape.svg)
	return err != nil && *parsedSvg == shape && balib.isShapeInCanvas(shape)
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
// LOCKS: Acquires and releases neighboursLock.
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

// Should be called whenever a new op is received, either from a blockartlib or another miner
// This functions:
// - validates the op
// - if valid, then adds the op to the currBlock, and then floods the op to other miners
// Returned error is nil if op is valid.
// @param op Op: Op to be validated.
// @return err error: nil if op is valid; otherwise can return one of the following errors:
//  	- InsufficientInkError
// 		- ShapeOverlapError
// 		- OutOfBoundsError
func receiveNewOp(op Op) (err error) {
	// acquire currBlock's lock
	blockLock.Lock()
	defer blockLock.Unlock()

	// check if op is valid
	if err = validateOp(op, currBlock); err != nil {
		// if not, return the error
		return err
	}

	// op is valid; add op to currBlock
	currBlock.ops = append(currBlock.ops, op)
	
	// floodOp on a separate thread; this miner's operation doesn't depend on the flood
	go floodOp(op)

	return nil
}

// Validates an operation. Returned error is nil if op is valid starting at 
// headBlock; false otherwise.
// - ASSUMES that if any locks are requred for headBlock, they have already been acquired
// @param op Op: Op to be validated.
// @param headBlock *Block: head block of chain from which ink will be calculated
// @return err error: nil if op is valid; otherwise can return one of the following errors:
//  	- InsufficientInkError
// 		- ShapeOverlapError
// 		- OutOfBoundsError
func validateOp(op Op, headBlock *Block) (err error) {
	if op.shape != nil {
		if e := validateShape(op.shape); e != nil {
			return e
		}
	}

	if op.shape != nil {
		// check if miner has enough ink only if op is owned by this miner
		// and if shape is not being deleted
		ink, err := balib.InkUsed(op.shape)
		inkAvail := inkAvail(op.owner, headBlock)
		if err != nil || inkAvail < ink {
			// not enough ink
			return balib.InsufficientInkError(inkAvail)
		}
	}

	if op.shape != nil {
		if hash := shapeOverlaps(op.shape, headBlock); hash != "" {
			// op is adding a shape that intersects with an already present shape; reject
			return balib.ShapeOverlapError(hash)
		}
	}

	if op.shape == nil {
		if e := shapeExists(op.shapeHash, op.owner, headBlock); e != nil {	
			// Op is trying to delete a shape that does not exist or that does not belong
			// to op.owner
			return e
		}
	}

	return nil
}

// TODO
// Checks if the passed shape is valid according to the spec
// Returned error is nil if shape is valid; otherwise, check the error
// - TODO shape fill spec re. convex or self-intersections
// - shape points are within the canvas
// @param shape *balib.Shape: pointer to shape that will be validated
// @return err error: Error indicating if shape is valid. Can be nil or one 
//                    of the following errors:
// 						- OutOfBoundsError
func validateShape(shape *balib.Shape) (err error) {
	// TODO
	return balib.OutOfBoundsError{}
}

// TODO
// - checks if the passed shape intersects with any shape currently on the canvas
//   that is NOT owned by this miner, starting at headBlock
// - ASSUMES that if any locks are requred for headBlock, they have already been acquired
// @param shape *balib.Shape: pointer to shape that will be checked for 
//                                  intersections
// @param headBlock *Block: head block of chain from which ink will be calculated
// @return shapeOverlapHash string: empty if shape does intersect with any other
//                                  non-owned shape; otherwise it is the hash of 
//                                  the shape this shape overlaps
func shapeOverlaps(shape *balib.Shape, headBlock *Block) (shapeOverlapHash string) {
	// TODO
	return ""
}

// TODO
// Checks if a shape with the given hash exists on the canvas (and was not later
// deleted) starting at headBlock, and that the passed owner is the owner of this shape
// Returned error is nil if shape does exist and is owned by owner, otherwise returns
// a non-nil error
// - ASSUMES that if any locks are requred for headBlock, they have already been acquired
// @param shapeHash string: hash of shape to check
// @param owner string: string identfying miner
// @param headBlock *Block: head block of chain from which ink will be calculated
// @return err error: Error indicating if shape is valid. Can be nil or one 
//                    of the following errors:
// 						- ShapeOwnerError
//						- TODO - error if shape does not exist?
func shapeExists(shapeHash string, ownder string, headBlock *Block) (err error) {
	// TODO
	return balib.ShapeOwnerError(shapeHash)
}

// TODO
// - counts the amount of ink currently available to passed miner starting at headBlock
// - ASSUMES that if any locks are requred for headBlock, they have already been acquired
// @param owner string: string identfying miner
// @param headBlock *Block: head block of chain from which ink will be calculated
// @return ink int: ink currently available to this miner, in pixels
func inkAvail(miner string, headBlock *Block) (ink int) {
	// TODO
	// Depends on starting ink, and how much ink you receive for each new block
	return 0
}

func main() {
	// ink-miner should take one parameter, which is its address
	// skip program
	args := os.Args[1:]

	numArgs := 1

	// check number of arguments
	if len(args) != numArgs {
		if len(args) < numArgs {
			fmt.Printf("too few arguments; expected %d, received%d\n", numArgs, len(args))
		} else {
			fmt.Printf("too many arguments; expected %d, received%d\n", numArgs, len(args))
		}
		// can't proceed without correct number of arguments
		return
	}

	address = args[0]

	// TODO - should communicate with server to get CanvasSettings and other miners in the network

	// Setup RPC
	server := rpc.NewServer()
	libMin := new(LibMin)
	server.Register(libMin)
	l, e := net.Listen("tcp", address)
	if e != nil {
		return
	}
	go server.Accept(l)
	
	// TODO - should start mining
}
