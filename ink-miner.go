/*
Implements the ink-miner for project 1 for UBC CS 416 2017 W2.

Usage:
$ go run ink-miner.go [client-incoming ip:port]

Example:
$ go run ink-miner.go 127.0.0.1:2020

*/

package main

import (
	"./blockartlib"
	"./proj1-server/rpcCommunication"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/md5"
	"crypto/rand"
	"encoding/gob"
	"encoding/hex"
	"fmt"
	"math/big"
	"net"
	"net/rpc"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
	//"crypto/x509"
	//"encoding/pem"
	"crypto/x509"
)

// Static
var publicKey ecdsa.PublicKey
var privateKey ecdsa.PrivateKey

// Current block
var currBlock *Block
var blockLock = &sync.Mutex{}

// Head block
var headBlockMeta *BlockMeta
var headBlockLock = &sync.Mutex{}

// Neighbours
var neighbours = make(map[net.Addr]InkMiner)
var neighboursLock = &sync.Mutex{}

// Network
var blockTree = make(map[string]*BlockMeta)
var serverConn *rpc.Client
var outgoingAddress string
var incomingAddress string

// Network Instructions
var minerNetSettings *rpcCommunication.MinerNetSettings

// slice of operation threads' channels that need to know about new blocks
var opChans = make(map[string](chan *BlockMeta))
var opChansLock = &sync.Mutex{}

type OpMeta struct {
	hash blockartlib.Hash
	r, s big.Int
	op   Op
}

type Op struct {
	shapeMeta       *blockartlib.ShapeMeta // not nil iff adding shape.
	deleteShapeHash string                 // non-empty iff removing shape.
	owner           ecdsa.PublicKey        // public key of miner that issued this op.
}

func (o Op) ToString() string {
	return fmt.Sprintf("%v", o)
}

type BlockMeta struct {
	hash  blockartlib.Hash
	r, s  big.Int // signature of the miner that mined this block.
	block Block
}

type Block struct {
	prev  blockartlib.Hash
	ops   []OpMeta
	len   int
	miner ecdsa.PublicKey // public key of the miner that mined this block.
	nonce string
}

func (b Block) ToString() string {
	return fmt.Sprintf("%+v", b)
}

type InkMiner struct {
	conn    *rpc.Client
	address net.Addr
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

type ServerConnectionError string

func (e ServerConnectionError) Error() string {
	return fmt.Sprintf("InkMiner: Could not connect to server for %s", string(e))
}

type KeyParseError string

func (e KeyParseError) Error() string {
	return fmt.Sprintf("InkMiner: Could not connect to server for %s", string(e))
}

type GensisBlockNotFound string

func (e GensisBlockNotFound) Error() string {
	return fmt.Sprintf("InkMiner: Could not find gensis block for %s", string(e))
}

type MinerSettingNotFound string

func (e MinerSettingNotFound) Error() string {
	return fmt.Sprintf("InkMiner: Could not find miner setting for %s", string(e))
}

// Receives op block flood calls. Verifies the op, which will add the op to currBlock and flood
// op if it is valid.
// @param op *Op: Op which will be verified, and potentially added and flooeded
// @param reply *bool: Bool indicating whether op was successfully validated
// @return err error: Any errors in receiving op.
func (m *MinMin) NotifyNewOp(opMeta *OpMeta, reply *bool) (err error) {
	*reply = true
	if e := receiveNewOp(*opMeta); e != nil {
		// validate was successful only if error is null
		*reply = false
		return e
	}
	return nil
}

// Receives block flood calls. Verifies chains. Updates head block if new chain is acknowledged.
// LOCKS: Acquires and releases headBlockLock
// @param blockMeta *BlockMeta: Block which was added to chain.
// @param reply *bool: Bool indicating success of RPC.
// @return error: Any errors produced during new block processing.
func (m *MinMin) NotifyNewBlock(blockMeta *BlockMeta, reply *bool) error {
	if b := blockTree[string(blockMeta.hash)]; b != nil {
		// We are already aware of this block.
		return nil
	}

	*reply = false

	// Verify chain.
	var inter interface{}
	if err := crawlChain(blockMeta, nil, inter, inter); err != nil {
		return err
	}

	*reply = true
	headBlockLock.Lock()
	defer headBlockLock.Unlock()
	if blockMeta.block.len > headBlockMeta.block.len {
		// head block is about to change; need to update currBlock
		blockLock.Lock()
		oldOps := currBlock.ops
		currBlock.ops = []OpMeta{}
		verificationChan := make(chan error, 1)
		for _, oldOp := range oldOps {
			// go through ops sequentially for simplicity
			// TODO - if runtime is really bad, could make it parallel
			// FIXME
			pseudoCurrBlockMeta := BlockMeta{block: *currBlock}
			go verifyOp(oldOp, &pseudoCurrBlockMeta, -1, verificationChan)
			err := <-verificationChan
			if err == nil {
				// op is still valid
				currBlock.ops = append(currBlock.ops, oldOp)
			}
		}
		close(verificationChan)

		currBlock.prev = blockMeta.hash
		blockLock.Unlock()

		// update headBlockMeta
		headBlockMeta = blockMeta
	}

	// notify all opChans
	for _, opChan := range opChans {
		go func() {
			opChan <- blockMeta
		}()
	}

	floodBlock(*blockMeta)

	return nil
}

// Called when miner has been given this miner as a neighbour, to notify this miner
// of its new neighbour.
// @param addr *net.Addr: address of calling miner.
// @param reply *bool: Bool indicating success of RPC.
// @return error: Any errors produced during new block processing.
func (m *MinMin) NotifyNewNeighbour(addr *net.Addr, reply *bool) error {
	inkMiner := addNeighbour(*addr)
	*reply = false
	if inkMiner != nil {
		*reply = true
		// send new neighbour this miner's headBlockMeta
		go inkMiner.conn.Call("MinMin.NotifyNewBlock", headBlockMeta, nil)

		// send currently pending ops
		for _, opMeta := range currBlock.ops {
			go inkMiner.conn.Call("MinMin.NotifyNewOp", &opMeta, nil)
		}
	}
	return nil
}

// Returns block identified with provided nonce.
// @param nonce *string: Nonce of block to be returned.
// @param block *Block: Pointer to block specified by nonce.
// @return error: Any errors produced in retrieval of block.
func (m *MinMin) RequestBlock(hash *[]byte, blockMeta *BlockMeta) error {
	blockMeta = blockTree[string(*hash)]
	if blockMeta == nil {
		return BlockNotFoundError(string(*hash))
	}
	return nil
}

// RPC for blockartlib-miner connection
type LibMin int

// Returns the CanvasSettings
// @param args int: required by Go's RPC; does nothing
// @param reply *blockartlib.ConvasSettings: pointer to CanvasSettings that will be returned
// @return error: Any errors produced
func (l *LibMin) OpenCanvas(args *blockartlib.OpenCanvasArgs, reply *blockartlib.OpenCanvasReply) (err error) {
	// Ensure art node has proper private & public keys.
	if args.Priv != privateKey || args.Pub != publicKey {
		return blockartlib.DisconnectedError("")
	}
	*reply = blockartlib.OpenCanvasReply{CanvasSettings: minerNetSettings.CanvasSettings}
	return nil
}

// Adds a new shape ot the canvas
// @param args *blockartlib.AddShapeArgs: contains the shape to be added, and the validateNum
// @param reply *blockartlib.AddShapeReply: pointer to AddShapeReply that will be returned
// @return error: Any errors produced
func (l *LibMin) AddShape(args *blockartlib.AddShapeArgs, reply *blockartlib.AddShapeReply) (err error) {
	// construct Op for shape
	op := Op{
		shapeMeta: &args.ShapeMeta,
		owner:     publicKey,
	}

	hash := hashOp(op)
	r, s, err := ecdsa.Sign(rand.Reader, &privateKey, hash)
	if err != nil {
		return err
	}

	opMeta := OpMeta{
		hash: hash,
		r:    *r,
		s:    *s,
		op:   op,
	}

	// set up channel for opReceiveNewBlock back result
	returnChan := make(chan error)

	// ensure hash is unique, even between add and delete shapes
	opChansKey := args.ShapeMeta.Hash + "a"

	// set up channel to receive new blocks
	opChansLock.Lock()
	opChan := make(chan *BlockMeta, 1)
	opChans[opChansKey] = opChan
	go opReceiveNewBlocks(opChan, returnChan, opMeta, args.ValidateNum)
	opChansLock.Unlock()

	defer func(opChan chan *BlockMeta, returnChan chan error, key string) {
		// clean up channels
		close(returnChan)
		delete(opChans, key)
		close(opChan)
	}(opChan, returnChan, opChansKey)

	// receiveNewOp will try to add opMeta to current block and flood opMeta
	if err = receiveNewOp(opMeta); err != nil {
		// return error in reply so that it is not cast
		reply.Error = err
		return nil
	}

	resultErr := <-returnChan
	if resultErr != nil {
		reply.Error = err
		return nil
	}

	reply.OpHash = hash.ToString()
	// Get ink
	getInkArgs := blockartlib.GetInkArgs{Miner: publicKey}
	return l.GetInk(&getInkArgs, &reply.InkRemaining)
}

// Returns the full SvgString for the given hash, if it exists locally, and even if it was later deleted
// Will not search the currBlock, only valid created blocks (no operation in currBlock will have returned yet,
// since validateNum >= 0, so those hashes will never be known to applications)
// @param args *blockartlib.GetSvgStringArgs: contains the hash of the shape to be returned
// @param reply *blockartlib.GetSvgStringReply: contains the shape string, and any internal errors
// @param err error: Any errors produced
func (l *LibMin) GetSvgString(args *blockartlib.GetSvgStringArgs, reply *blockartlib.GetSvgStringReply) (err error) {
	// Search for shape in set of local blocks
	// NOTE: as per https://piazza.com/class/jbyh5bsk4ez3cn?cid=425,
	// do not search externally; assume that any external blocks will get
	// flooded to this miner soon.
	opMeta := findOpMeta(args.OpHash)
	if opMeta == nil {
		// shape does not exist, return InvalidShapeHashError
		reply.Error = blockartlib.InvalidShapeHashError(args.OpHash)
		return nil
	}

	shapeMeta := opMeta.op.shapeMeta
	var stroke, fill string
	if shapeMeta != nil {
		// this is an add op
		stroke = shapeMeta.Shape.BorderColor
		fill = shapeMeta.Shape.FillColor
	}
	if shapeMeta == nil {
		// this is a delete op
		// find the shape by hash
		shapeMeta = findShapeMeta(opMeta.op.deleteShapeHash)
		if shapeMeta == nil {
			// shape does not exist, return InvalidShapeHashError
			reply.Error = blockartlib.InvalidShapeHashError(args.OpHash)
			return nil
		}

		// return a white shape, to delete the shape
		stroke = "white"
		fill = "white"
	}

	// Return html-valid tag, of the form:
	// <path d=[svgString] stroke=[stroke] fill=[fill]/>
	reply.SvgString = fmt.Sprintf("<path d=\"%s\" stroke=\"%s\" fill=\"%s\"/>", shapeMeta.Shape.Svg, stroke, fill)
	reply.Error = nil
	return nil
}

// Returns the amount of ink remaining for this miner, in pixels
// @param args args *int: dummy argument that is not used
// @param reply *uint32: amount of remaining ink, in pixels
// @param err error: Any errors produced
func (l *LibMin) GetInk(args *blockartlib.GetInkArgs, reply *uint32) (err error) {
	// acquire currBlock's lock
	blockLock.Lock()
	defer blockLock.Unlock()

	*reply = inkAvailCurr()
	return nil
}

// Deletes the shape associated with the passed deleteShapeHash, if it exists and is owned by this miner.
// args.ValidateNum specifies the number of blocks (no-op or op) that must follow the block with this
// operation in the block-chain along the longest path before the operation can return successfully.
// @param args *blockartlib.DeleteShapeArgs: contains the ValidateNum and ShapeHash
// @param reply *blockartlib.DeleteShapeReply: contains the ink remaining, and any internal errors
// @param err error: Any errors produced
func (l *LibMin) DeleteShape(args *blockartlib.DeleteShapeArgs, reply *blockartlib.DeleteShapeReply) (err error) {
	// construct Op for deletion
	op := Op{
		deleteShapeHash: args.ShapeHash,
		owner:           publicKey,
	}

	hash := hashOp(op)
	r, s, err := ecdsa.Sign(rand.Reader, &privateKey, hash)
	if err != nil {
		return err
	}

	opMeta := OpMeta{
		hash: hash,
		r:    *r,
		s:    *s,
		op:   op,
	}

	// set up channel for opReceiveNewBlock back result
	returnChan := make(chan error)

	// ensure hash is unique, even between add and delete shapes
	opChansKey := args.ShapeHash + "d"

	// set up channel to receive new blocks
	opChansLock.Lock()
	opChan := make(chan *BlockMeta, 1)
	opChans[opChansKey] = opChan
	go opReceiveNewBlocks(opChan, returnChan, opMeta, args.ValidateNum)
	opChansLock.Unlock()

	defer func(opChan chan *BlockMeta, returnChan chan error, key string) {
		// clean up channels
		close(returnChan)
		delete(opChans, key)
		close(opChan)
	}(opChan, returnChan, opChansKey)

	// receiveNewOp will try to add op to current block and flood op
	if err = receiveNewOp(opMeta); err != nil {
		// return error in reply so that it is not cast
		reply.Error = blockartlib.ShapeOwnerError(args.ShapeHash)
		return nil
	}

	resultErr := <-returnChan
	if resultErr != nil {
		reply.Error = blockartlib.ShapeOwnerError(args.ShapeHash)
		return nil
	}

	// Get ink
	getInkArgs := blockartlib.GetInkArgs{Miner: publicKey}
	return l.GetInk(&getInkArgs, &reply.InkRemaining)
}

// Returns the shape hashes contained by the block in BlockHash
// NOTE: as per https://piazza.com/class/jbyh5bsk4ez3cn?cid=425,
// do not search externally; assume that any external blocks will get
// flooded to this miner soon.
// @param args *string: the blockHash
// @param reply *blockartlib.GetShapesReply: contains the slice of shape hashes and any internal errors
// @param err error: Any errors produced
func (l *LibMin) GetShapes(args *string, reply *blockartlib.GetShapesReply) (err error) {
	// Search for block locally - if it does not exist, return an InvalidBlockHashError
	blockMeta, ok := blockTree[*args]
	if !ok || blockMeta == nil {
		// block does not exist locally
		reply.Error = blockartlib.InvalidBlockHashError(*args)
		return nil
	}

	for _, opMeta := range blockMeta.block.ops {
		// add op's hash to reply.ShapeHashes
		hash := opMeta.op.deleteShapeHash
		if opMeta.op.shapeMeta != nil {
			hash = opMeta.op.shapeMeta.Hash
		}
		reply.ShapeHashes = append(reply.ShapeHashes, hash)
	}

	reply.Error = nil
	return nil
}

// Returns the hash of the genesis block of the block chain
// @param args args *int: dummy argument that is not used
// @param reply *uint32: hash of genesis block
// @param err error: Any errors produced
func (l *LibMin) GetGenesisBlock(args *int, reply *blockartlib.Hash) (err error) {
	if minerNetSettings.GenesisBlockHash == blockartlib.Hash([]byte{}).ToString() {
		return GensisBlockNotFound("")
	}
	*reply, _ = hex.DecodeString(minerNetSettings.GenesisBlockHash)
	return nil
}

// Returns the shape hashes contained by the block in BlockHash
// NOTE: as per https://piazza.com/class/jbyh5bsk4ez3cn?cid=425,
// do not search externally; assume that any external blocks will get
// flooded to this miner soon.
// @param args *[]byte: the blockHash
// @param reply *blockartlib.GetChildrenReply: contains the slice of block hashes and any internal errors
// @param err error: Any errors produced
func (l *LibMin) GetChildren(args *[]byte, reply *blockartlib.GetChildrenReply) (err error) {
	// First, see if block exists locally
	if blockMeta, ok := blockTree[string(*args)]; !ok || blockMeta == nil {
		// block does not exist locally
		reply.Error = blockartlib.InvalidBlockHashError(*args)
		return nil
	}

	// If it exists, then just search for children whose parent is the passed BlockHash
	for hash, blockMeta := range blockTree {
		if string(blockMeta.block.prev) == string(*args) {
			reply.BlockHashes = append(reply.BlockHashes, hash)
		}
	}

	reply.Error = nil
	return nil
}

// This helper function receives information about new blocks for the purpose of ensuring an operation
// is successfully added to the blockchain
// @param opChan: channel through which new blocks will be sent
// @param returnChan: channel through which the result of this function should be sent
// @param opMeta: the opMeta we're trying to get added to the blockchain
// @param validateNum: the number of blocks required after a block containing opMeta in the blockchain
//                     for the add to ba success
func opReceiveNewBlocks(opChan chan *BlockMeta, returnChan chan error, opMeta OpMeta, validateNum uint8) {
	for {
		blockMeta := <-opChan
		// idea - see if opMeta appears in the chain for this block
		// if it does, check that validateNum number of blocks have been added on top
		// if it is not, and this is the new head, resend the block
		cur := blockMeta
		// can iterate through chain because block has already been validated
		foundOp := false

	chainCrawl:
		for !isGenesis(*cur) {
			for _, opIter := range cur.block.ops {
				if opMetasEqual(opIter, opMeta) {
					// found the opMeta in this chain
					foundOp = true
					if (blockMeta.block.len - cur.block.len) >= int(validateNum) {
						// enough blocks have been added
						returnChan <- nil
						return
					}

					break chainCrawl
				}
			}

			var ok bool
			if cur, ok = blockTree[cur.block.prev.ToString()]; !ok {
				// chain should have been valid, this should never happen
				// just ignore this block
				break chainCrawl
			}
		}

		if !foundOp && blockMetasEqual(*blockMeta, *headBlockMeta) {
			// opMeta is not in the longest chain; resend the opMeta and flood it
			if err := receiveNewOp(opMeta); err != nil {
				// new longest chain now has a conflict with the
				// return error in reply so that it is not cast
				returnChan <- err
				return
			}
		}
	}
}

// Compares two OpMetas, and returns true if they are equal
// @param block1: the first OpMeta to compare
// @param block2: the second OpMeta to compare
// @return bool: true if the OpMetas are equal, false otherwise
func opMetasEqual(opMeta1 OpMeta, opMeta2 OpMeta) bool {
	return opMeta1.hash.ToString() == opMeta2.hash.ToString() && opMeta1.r.Cmp(&opMeta2.r) == 0 && opMeta1.s.Cmp(&opMeta2.s) == 0 && opMeta1.op == opMeta2.op
}

// Compares two blockMetas, and returns true if they are equal
// @param block1: the first blockMeta to compare
// @param block2: the second blockMeta to compare
// @return bool: true if the blockMetas are equal, false otherwise
func blockMetasEqual(blockMeta1 BlockMeta, blockMeta2 BlockMeta) bool {
	if blockMeta1.hash.ToString() != blockMeta2.hash.ToString() || blockMeta1.r.Cmp(&blockMeta2.r) != 0 || blockMeta1.s.Cmp(&blockMeta2.s) != 0 {
		return false
	}

	return blocksEqual(blockMeta1.block, blockMeta2.block)
}

// Compares two blocks, and returns true if they are equal
// For block.ops, the operations must be in the same order
// @param block1: the first block to compare
// @param block2: the second block to compare
// @return bool: true if the blocks are equal, false otherwise
func blocksEqual(block1 Block, block2 Block) bool {
	if block1.prev.ToString() != block2.prev.ToString() || block1.len != block2.len || block1.nonce != block2.nonce || len(block1.ops) != len(block2.ops) {
		return false
	}

	for i := 0; i < len(block1.ops); i++ {
		if !opMetasEqual(block1.ops[i], block2.ops[i]) {
			return false
		}
	}

	return true
}

// Searches for an opMeta in the set of local blocks with the given hash.
// @param opHash string: hash of opMeta that is being searched for
// @return shape: found op whose hash matches opHash; nil if it does not exist
func findOpMeta(opHash string) (opMeta *OpMeta) {
	// Iterate through all locally stored blocks to search for a shape with the passed hash
	for _, blockMeta := range blockTree {
		block := blockMeta.block
		// search through the block's ops
		for _, opMeta := range block.ops {
			if opMeta.hash.ToString() == opHash {
				// opMeta was found
				return &opMeta
			}
		}
	}

	// opMeta was not found
	return nil
}

// Searches for a shapeMeta with the given hash in the set of add ops in local blocks.
// @param shapeHash string: hash of shapeMeta that is being searched for
// @return shapeMeta: found shapeMeta whose hash matches shapeHash; nil if it does not
//                    exist
func findShapeMeta(shapeHash string) (shapeMeta *blockartlib.ShapeMeta) {
	// Iterate through all locally stored blocks to search for a shape with the passed hash
	for _, blockMeta := range blockTree {
		block := blockMeta.block
		// search through the block's ops
		for _, opMeta := range block.ops {
			if opMeta.op.shapeMeta != nil && opMeta.op.shapeMeta.Hash == shapeHash {
				// shapeMeta was found
				return opMeta.op.shapeMeta
			}
		}
	}

	// shapeMeta was not found
	return nil
}

// Runs the passed function on each element in the blockchain (including the headBlock),
// starting at headBlock.
// To do this, crawlChain first builds up the entire chain and validates any external blocks
// in reverse order. If any block is not valid, returns the error for the entire crawl and does not
// store any blocks.
// If all blocks are valid, then makes a third pass through, calling fn on each block, starting at
// headBlock. If fn returns an error, stops iterating and returns the error
// - ASSUMES that if any locks are requred for headBlock, they have already been acquired
// @param headBlock *Block: head block of chain from which ink will be calculated
// @param fn func: the helper function that will be run on each block. It behaves like an RPC call, and
//                 takes 3 arguments and returns an error:
//				   @param *Block: the block on which the function is called
//				   @param interface: the arguments to the function (like in RPC)
//				   @param interface: the reply from the function; MUST be a pointer to a struct
//									 (return values, again like in RPC)
//				   @return bool: whether the function is done or not; if true, will return to caller,
//                               otherwise, continues onto the next chain in the blockchain
//				   @return error: any errors returned by the function
//				   Note that the args and reply must have a struct defintion (like in RPC) and must
//                 be cast to that type in fn, with a call like argsT, ok := args.(Type)
// @return err error: returns any errors encountered, orone of the following errors:
// 		- InvalidBlockHashError
func crawlChain(headBlock *BlockMeta, fn func(*BlockMeta, interface{}, interface{}) (bool, error), args interface{}, reply interface{}) (err error) {
	if fn == nil {
		fn = crawlNoopHelper
	}

	// the chain, starting at headBlock
	chain := []*BlockMeta{}
	curr := headBlockMeta
	for {
		// add current element to the end of the chain
		chain = append(chain, curr)
		parent := crawlChainHelperGetBlock(curr.block.prev)
		if parent == nil {
			// If the parent could not be found, then the hash is invalid.
			return blockartlib.InvalidBlockHashError(string(curr.hash))
		}

		if isGenesis(*curr) {
			// We are at the end of the chain.
			break
		}

		curr = parent
	}

	// Validate in reverse order (from GenesisBlock to headBlock).
	for i := len(chain) - 1; i >= 0; i-- {
		blockMeta := chain[i]
		if _, exists := blockTree[string(blockMeta.hash)]; exists {
			// Block is already stored locally, so has already been validated
			continue
		} else {
			// validate block, knowing that all parent blocks are valid
			if err = validateBlock(chain[i:]); err != nil {
				// The block was not valid, return the error.
				return err
			}

			// Block is valid, so add it to the map.
			blockTree[string(blockMeta.hash)] = blockMeta
		}
	}

	// Blocks are valid, so now run the function on each block in the chain,
	// starting from the headBlock.
	for _, blockMeta := range chain {
		done, err := fn(blockMeta, args, reply)
		if err != nil || done {
			// if fn is done, or there is an error, return
			return err
		}
	}

	return nil
}

// No-op crawl helper
// This function should be used when the default behaviour of crawlChain is sufficient
// @param: block: block on which the function is called; does nothing
// @param: args: unused
// @param: reply: unused
// @return done bool: returns true, since there is no more work to do
// @return err error: always nil
func crawlNoopHelper(blockMeta *BlockMeta, args interface{}, reply interface{}) (done bool, err error) {
	return true, nil
}

// Returns block with given hash.
// If the block is not stored locally, try to get the block from another miner.
// NOTE: this operation does no verification on any external blocks.
// @param hash blockartlib.Hash: The hash of the block to get info on.
// @return Block: The requested block, or nil if no block is found.
func crawlChainHelperGetBlock(hash blockartlib.Hash) (blockMeta *BlockMeta) {
	// Search locally.
	if blockMeta, ok := blockTree[hash.ToString()]; ok && blockMeta != nil {
		return blockMeta
	}

	// block is not stored locally, search externally.
	for _, n := range neighbours {
		err := n.conn.Call("MinMin.RequestBlock", hash, blockMeta)
		if err != nil {
			// Block not found, keep searching.
			continue
		}
		// return the block
		return blockMeta
	}

	// Block not found.
	return nil
}

// Validates the FIRST block in the slice, ASSUMING that all other blocks in the
// chain have already been validated
// @param chain []*Block: the block chain. The first element in the slice is the
//                        block being validated, assume rest of blocks are valid
//                        (and thus the last block should be the Genesis block)
// @return err error: any errors from validation; nil if block is valid
func validateBlock(chain []*BlockMeta) (err error) {
	blockMeta := *chain[0]

	// Verify hash.
	if hashBlock(blockMeta.block).ToString() != blockMeta.hash.ToString() {
		return blockartlib.InvalidBlockHashError(blockMeta.hash)
	}
	// Verify block signature.
	if !ecdsa.Verify(&blockMeta.block.miner, blockMeta.hash, &blockMeta.r, &blockMeta.s) {
		return blockartlib.InvalidBlockHashError(string(blockMeta.hash))
	}
	// Verify POW.
	if err = verifyBlockNonce(blockMeta.block.nonce, len(blockMeta.block.ops) == 0); err != nil {
		return err
	}
	// Verify ops.
	if err = verifyOps(blockMeta.block); err != nil {
		return err
	}

	return nil
}

// Returns true if block is the genesis block.
// @param block Block: The block to test against.
// @return bool: True iff block is genesis block.
func isGenesis(blockMeta BlockMeta) bool {
	block := blockMeta.block
	return block.prev.ToString() == "" && hashBlock(block).ToString() == minerNetSettings.GenesisBlockHash
}

// Returns hash of block.
// @param block Block: Block to be hashed.
// @return Hash: The hash of the block.
func hashBlock(block Block) blockartlib.Hash {
	hasher := md5.New()
	hasher.Write([]byte(block.ToString()))
	return blockartlib.Hash(hasher.Sum(nil)[:])
}

// Returns hash of op.
// @param op Op: Op to be hashed.
// @return Hash: The hash of the op.
func hashOp(op Op) blockartlib.Hash {
	hasher := md5.New()
	hasher.Write([]byte(op.ToString()))
	return blockartlib.Hash(hasher.Sum(nil)[:])
}

// Returns hash of string.
// @param s string: The string to hash.
// @return []byte: The hash of the string.
func hashString(s string) []byte {
	hasher := md5.New()
	hasher.Write([]byte(s))
	return hasher.Sum(nil)[:]
}

// Verifies that hash meets POW requirements specified by server.
// @param hash string: Hash of block to be verified.
// @return bool: True iff valid.
func verifyBlockNonce(hash string, noop bool) error {
	pow := minerNetSettings.PoWDifficultyOpBlock
	if noop {
		pow = minerNetSettings.PoWDifficultyNoOpBlock
	}
	n := int(pow)
	if hash[len(hash)-n:] == strings.Repeat("0", n) {
		return nil
	}
	return blockartlib.InvalidBlockHashError(hash)
}

// Verifies that all ops are valid and no shape conflicts exist against blockchain canvas.
// @param ops []Op: Slice of ops to verify.
// @return error: nil iff valid.
func verifyOps(block Block) error {
	verificationChan := make(chan error, 1)

	pseudoBlockMeta := BlockMeta{block: block}
	for i, opMeta := range block.ops {
		go verifyOp(opMeta, &pseudoBlockMeta, i, verificationChan)
	}

	pendingVerifications := len(block.ops)
	for pendingVerifications != 0 {
		err := <-verificationChan
		if err != nil {
			return err
		}
	}

	return nil
}

// Verifies an op against all ops in the blockchain starting at blockMeta. Assumes all previous blocks in
// chain are valid. Skip the operation itself for validation.
// ASSUME: any blocks required for blockMeta have already been acquired
// @param candidateOp Op: The op to verify.
// @param blockMeta *Block: The starting blockMeta on which to begin the verification.
//							Note: this does not need to be a fully valid block; it can also be a pseudo block meta
//                                created around currBlock
// @param ch chan<-error: The channel to which verification errors are piped into, or nil if no errors are found.
func verifyOp(candidateOpMeta OpMeta, blockMeta *BlockMeta, indexInBlock int, ch chan<- error) {
	// Verify hash.
	if hashOp(candidateOpMeta.op).ToString() != candidateOpMeta.hash.ToString() {
		ch <- blockartlib.InvalidShapeHashError(candidateOpMeta.hash.ToString())
		return
	}

	candidateOp := candidateOpMeta.op

	// Verify signature.
	if !ecdsa.Verify(&candidateOp.owner, candidateOpMeta.hash, &candidateOpMeta.r, &candidateOpMeta.s) {
		ch <- blockartlib.InvalidShapeHashError(candidateOpMeta.hash.ToString())
		return
	}

	if candidateOp.shapeMeta != nil {
		// Verify shape.
		if err := validateShape(candidateOp.shapeMeta); err != nil {
			ch <- err
			return
		}

		// Verify op with shape.
		shape := candidateOp.shapeMeta.Shape
		// Ensure svg string isn't beyond the maximum specified length.
		if svg := shape.Svg; blockartlib.IsSvgTooLong(svg) {
			ch <- blockartlib.ShapeSvgStringTooLongError(svg)
			return
		}

		// Ensure shape is on the canvas.
		if !blockartlib.IsShapeInCanvas(shape) {
			ch <- blockartlib.OutOfBoundsError{}
			return
		}

		// Ensure miner has enough ink.
		inkAvail := inkAvail(candidateOp.owner, blockMeta)
		if indexInBlock >= 0 {
			// op is in the block, so don't double count the ink it uses
			inkAvail -= shape.Ink
		}
		if inkAvail < shape.Ink {
			ch <- blockartlib.InsufficientInkError(inkAvail)
			return
		}

		// Ensure op is not duplicate and shape does not overlap with other ops in the chain.
		curr := blockMeta
		for {
			for i, opMeta := range curr.block.ops {
				// test if curr == blockMeta
				// aren't guaranteed blockMeta is a valid meta, just use block
				if curr.block.prev.ToString() == blockMeta.block.prev.ToString() && i == indexInBlock {
					// this is the op itself in the block; skip it
					continue
				}

				// This op has been performed before.
				if candidateOpMeta.hash.ToString() == opMeta.hash.ToString() {
					ch <- blockartlib.OutOfBoundsError{}
					return
				}
				if candidateOpMeta.op.owner != opMeta.op.owner {
					if blockartlib.ShapesIntersect(shape, opMeta.op.shapeMeta.Shape, minerNetSettings.CanvasSettings) {
						ch <- blockartlib.ShapeOverlapError(candidateOpMeta.op.shapeMeta.Hash)
						return
					}
				}
			}

			// Exit loop once we verify no overlap conflicts in the genesis block.
			if isGenesis(*curr) {
				break
			}

			var ok bool
			curr, ok = blockTree[curr.block.prev.ToString()]
			if !ok {
				ch <- blockartlib.InvalidBlockHashError(curr.block.prev.ToString())
				return
			}
		}
	} else {
		// TODO: Return error if already encountered @Matthew
		// op is a delete; verify shape existed on the canvas, and belonged to this miner
		curr := blockMeta
		for {
			for i, opMeta := range curr.block.ops {
				// only want to search through ops that appear *before* this op, so if i == -1, that's all ops
				// and if i >= 0, that's all ops with a *smaller* index
				// test if curr == blockMeta, and that the
				// aren't guaranteed blockMeta is a valid meta, just use block
				if curr.block.prev.ToString() == blockMeta.block.prev.ToString() && indexInBlock >= 0 && i >= indexInBlock {
					// this op is after candidateOpMeta
					continue
				}

				if opMeta.op.shapeMeta != nil && opMeta.op.shapeMeta.Hash == opMeta.op.deleteShapeHash {
					// found the op for adding the shape
					if opMeta.op.owner == candidateOp.owner {
						// correct owner
						ch <- nil
						return
					}

					// incorrect owner
					ch <- blockartlib.ShapeOwnerError(candidateOp.deleteShapeHash)
					return
				}
			}

			// Exit loop once we verify no overlap conflicts in the genesis block.
			if isGenesis(*curr) {
				break
			}

			var ok bool
			curr, ok = blockTree[curr.block.prev.ToString()]
			if !ok {
				ch <- blockartlib.InvalidBlockHashError(curr.block.prev.ToString())
				return
			}
		}

		// could not find shape
		ch <- blockartlib.ShapeOwnerError(candidateOp.deleteShapeHash)
		return
	}

	ch <- nil
	return
}

// Sends op to all neighbours.
// LOCKS: Calls neighboursLock.Lock().
// @param opMeta OpMeta: Op to be broadcast.
func floodOp(opMeta OpMeta) {
	// Prevent other processes from adding/removing neighbours.
	neighboursLock.Lock()
	defer neighboursLock.Unlock()

	replies := 0
	replyChan := make(chan *rpc.Call, 1)

	for _, n := range neighbours {
		var reply bool
		_ = n.conn.Go("NotifyNewOp", opMeta, &reply, replyChan)
	}

	for replies != len(neighbours) {
		select {
		case <-replyChan:
			replies++
		case <-time.After(2 * time.Second):
			replies++
		}
	}
}

// Sends block to all neighbours.
// LOCKS: Acquires and releases neighboursLock.
// @param block Block: Block to be broadcast.
func floodBlock(blockMeta BlockMeta) {
	// Prevent other processes from adding/removing neighbours.
	neighboursLock.Lock()
	defer neighboursLock.Unlock()

	replies := 0
	replyChan := make(chan *rpc.Call, 1)

	for _, n := range neighbours {
		var reply bool
		_ = n.conn.Go("NotifyNewBlock", blockMeta, &reply, replyChan)
	}

	for replies != len(neighbours) {
		select {
		case <-replyChan:
			replies++
		case <-time.After(2 * time.Second):
			replies++
		}
	}
}

// Should be called whenever a new op is received, either from a blockartlib or another miner
// This functions:
// - validates the op
// - if valid, then adds the op to the currBlock, and then floods the op to other miners
// Returned error is nil if op is valid.
// @param opMeta OpMeta: Op to be validated.
// @return err error: nil if op is valid; otherwise can return one of the following errors:
//  	- InsufficientInkError
// 		- ShapeOverlapError
// 		- OutOfBoundsError
func receiveNewOp(opMeta OpMeta) (err error) {
	// acquire currBlock's lock
	blockLock.Lock()
	defer blockLock.Unlock()

	// check if op is valid
	verifyCh := make(chan error, 1)
	pseudoCurrBlockMeta := BlockMeta{block: *currBlock}
	go verifyOp(opMeta, &pseudoCurrBlockMeta, -1, verifyCh)

	err = <-verifyCh
	if err != nil {
		return err
	}

	// op is valid; add op to currBlock
	currBlock.ops = append(currBlock.ops, opMeta)

	// floodOp on a separate thread; this miner's operation doesn't depend on the flood
	go floodOp(opMeta)

	return nil
}

// Checks if the passed shape is valid according to the spec
// Returned error is nil if shape is valid; otherwise, check the error
// - shape points are within the canvas
// @param candidateShapeMeta *blockartlib.ShapeMeta: Shape that will be validated.
// @return err error: Error indicating if shape is valid.
func validateShape(candidateShapeMeta *blockartlib.ShapeMeta) (err error) {
	candidateShape := candidateShapeMeta.Shape

	// Ensure hash is correct.
	if candidateShapeMeta.Hash != blockartlib.HashShape(candidateShape) {
		return blockartlib.OutOfBoundsError{}
	}

	// Ensure shape properties correspond to the svg path.
	shape, err := blockartlib.ParseSvgPath(candidateShape.Svg)
	if err != nil {
		return err
	}

	if shape.FilledIn != candidateShape.FilledIn {
		return blockartlib.OutOfBoundsError{}
	}

	if shape.FillColor != candidateShape.FillColor {
		return blockartlib.OutOfBoundsError{}
	}

	if shape.BorderColor != candidateShape.BorderColor {
		return blockartlib.OutOfBoundsError{}
	}

	if len(shape.Edges) != len(candidateShape.Edges) {
		return blockartlib.OutOfBoundsError{}
	}

	candidateEdges := candidateShape.Edges
	sort.Sort(blockartlib.Edges(candidateEdges))
	edges := shape.Edges
	sort.Sort(blockartlib.Edges(shape.Edges))

	for _, e1 := range edges {
		for _, e2 := range candidateEdges {
			if e1 != e2 {
				return blockartlib.OutOfBoundsError{}
			}
		}
	}

	// Ensure accuracy of Ink parameter.
	ink, err := blockartlib.InkUsed(shape)
	if err != nil {
		return err
	}

	if ink != candidateShape.Ink {
		return blockartlib.OutOfBoundsError{}
	}

	if !blockartlib.IsSimpleShape(shape) {
		return blockartlib.OutOfBoundsError{}
	}

	return blockartlib.OutOfBoundsError{}
}

///////////////////////////////////////////////////////////
/* Structs and helper function for crawlChain for getInk */
///////////////////////////////////////////////////////////
type inkAvailCrawlArgs struct {
	miner ecdsa.PublicKey
}

type inkAvailCrawlReply struct {
	removedShapeHashes []string
	ink                uint32
}

// Checks the block for a shape with the passed args.deleteShapeHash
// If the shape was ever added, set reply.shape to the shape.
// Also count how many times the shape was added
// This function should be used when the default behaviour of crawlChain is sufficient
// @param: blockMeta *BlockMeta: block on which the function is called; does nothing
// @param: args interface{]: a inkAvailCrawlArgs contianing the miner whose remaining ink we're finding
// @param: reply interface{}: a inkAvailCrawlReply that will contain the ink remaining
// @return done bool: wehther the shape has been found (whether deleted or not, since the search is
//                    done in both situations)
// @return err error: any errors encountered
func inkAvailCrawlHelper(blockMeta *BlockMeta, args interface{}, reply interface{}) (done bool, err error) {
	crawlArgs, ok := args.(inkAvailCrawlArgs)
	if !ok {
		// args is invalid; return an error
		return true, blockartlib.DisconnectedError("")
	}

	crawlReply, ok := reply.(inkAvailCrawlReply)
	if !ok {
		// reply is invalid; return an error
		return true, blockartlib.DisconnectedError("")
	}

	// for simplicity, iterate through the block twice
	// first, look only for delete operations
	for _, opMeta := range blockMeta.block.ops {
		op := opMeta.op
		if op.deleteShapeHash != "" && op.owner == crawlArgs.miner {
			// shape was removed and owned by this miner
			crawlReply.removedShapeHashes = append(crawlReply.removedShapeHashes, op.deleteShapeHash)
		}
	}

	// second, look only for added operations
	for _, opMeta := range blockMeta.block.ops {
		op := opMeta.op
		if op.shapeMeta != nil && op.owner == crawlArgs.miner {
			// shape was added and owned by this miner
			index := searchSlice(op.shapeMeta.Hash, crawlReply.removedShapeHashes)
			if index >= 0 && index < len(crawlReply.removedShapeHashes) {
				// shape was later removed; do not charge for ink
				// remove deleteShapeHash from the list of removedShapeHashes
				crawlReply.removedShapeHashes = append(crawlReply.removedShapeHashes[:index], crawlReply.removedShapeHashes[index+1:]...)
			} else {
				// shape was not removed
				crawlReply.ink -= op.shapeMeta.Shape.Ink
			}
		}
	}

	// TODO - add ink if crawlArgs.miner mined this block

	// Continue searching down the chain
	return false, nil
}

// Searches for search in slice of strings
// @param search: element you're looking for
// @param slice: slice you're searching over
// @return index int: index of search in slice if it exists, otherwise -1
func searchSlice(search string, slice []string) (index int) {
	for i, s := range slice {
		if s == search {
			return i
		}
	}
	return -1
}

// Counts the amount of ink currently available to passed miner starting at headBlock
// - ASSUMES that if any locks are requred for headBlock, they have already been acquired
// @param owner ecdsa.PublicKey: public key identfying miner
// @param headBlock *Block: head block of chain from which ink will be calculated
// @return ink uint32: ink currently available to the miner, in pixels
func inkAvail(miner ecdsa.PublicKey, headBlock *BlockMeta) (ink uint32) {
	// the crawl by default does all the work we need, so no special helper/args/reply is required
	args := &inkAvailCrawlArgs{miner: miner}
	var reply inkAvailCrawlReply

	if err := crawlChain(headBlock, inkAvailCrawlHelper, args, &reply); err != nil {
		// error while searching; just return 0
		return 0
	}

	return reply.ink
}

// Counts the amount of ink currently available to this miner starting at currBlock
// @return ink uint32: ink currently available to this miner, in pixels
func inkAvailCurr() (ink uint32) {
	// the crawl by default does all the work we need, so no special helper/args/reply is required
	args := &inkAvailCrawlArgs{miner: publicKey}
	var reply inkAvailCrawlReply

	// first count up the amount of ink used in currBlock
	// look only for delete operations
	for _, opMeta := range currBlock.ops {
		op := opMeta.op
		if op.deleteShapeHash != "" && op.owner == publicKey {
			// shape was removed and owned by this miner
			reply.removedShapeHashes = append(reply.removedShapeHashes, op.deleteShapeHash)
		}
	}
	// look only for added operations
	for _, opMeta := range currBlock.ops {
		op := opMeta.op
		if op.shapeMeta != nil && op.owner == publicKey {
			// shape was added and owned by this miner
			index := searchSlice(op.shapeMeta.Hash, reply.removedShapeHashes)
			if index >= 0 && index < len(reply.removedShapeHashes) {
				// shape was later removed; do not charge for ink
				// remove deleteShapeHash from the list of removedShapeHashes
				reply.removedShapeHashes = append(reply.removedShapeHashes[:index], reply.removedShapeHashes[index+1:]...)
			} else {
				// shape was not removed
				reply.ink -= op.shapeMeta.Shape.Ink
			}
		}
	}

	// second, go through the rest of the chain
	if err := crawlChain(headBlockMeta, inkAvailCrawlHelper, args, &reply); err != nil {
		// error while searching; just return 0
		return 0
	}

	return reply.ink
}

/*
	Registering the miner to the server, calling the server's RPC method
	@return error: ServerConnectionError if connection to server fails
*/
func registerMinerToServer() error {
	tcpAddr, err := net.ResolveTCPAddr("tcp", incomingAddress)
	if err != nil {
		return ServerConnectionError("resolve tcp error")
	}
	minerSettings := rpcCommunication.MinerInfo{Address: tcpAddr, Key: publicKey}
	clientErr := serverConn.Call("RServer.Register", minerSettings, &minerNetSettings)
	if clientErr != nil {
		fmt.Println(clientErr)
		return ServerConnectionError("registration failure ")
	}
	return nil
}

/*
	After registering with the server, the miner will ping the server every
	specified interval / 2
	@return error: ServerConnectionError if connection to server fails
*/
func startHeartBeat() error {
	frequency := time.Duration(minerNetSettings.HeartBeat/4) * time.Millisecond
	for {
		var reply bool
		// passing the miners public key and a dummy reply
		clientErr := serverConn.Call("RServer.HeartBeat", publicKey, &reply)
		if clientErr != nil {
			// TODO ->
			fmt.Println(clientErr)
			return clientErr
		}

		time.Sleep(frequency)
	}

	return nil
}

/*
	TODO: checking errors -> can we see what errors the server returns
	Request nodes from the server, will add a neighbouring ink miner , or throw a disconnected error
	@return: Server disconnected errors for rpc failures
*/
func getNodes() error {
	var newNeighbourAddresses []net.Addr
	clientErr := serverConn.Call("RServer.GetNodes", &publicKey, &newNeighbourAddresses)
	if clientErr != nil {
		return ServerConnectionError("get nodes failure")
	}

	neighboursLock.Lock()
	for _, address := range newNeighbourAddresses {
		if inkMiner := addNeighbour(address); inkMiner != nil {
			// notify new neighbours that you are their neighbour
			var reply bool
			inkMiner.conn.Call("MinMin.NotifyNewNeighbour", &address, &reply)
			if !reply {
				// remove neighbour from map
				delete(neighbours, address)
			}
		}
	}
	neighboursLock.Unlock()
	return nil
}

/*
	Tries to add neighbour to local slice of neighbours
	NOTE: requires neighboursLock to be acquired before calling this function
	@param: outgoing address of the new neighbour
	@return: InkMiner of added neighbour, nil if neighbour was not added
*/
func addNeighbour(address net.Addr) *InkMiner {
	// only add it if the neighbor does not already exist
	if !doesNeighbourExist(address) {
		client, err := rpc.Dial(address.Network(), address.String())
		if err != nil {
			// can not connect to a node
			return nil
		}

		inkMiner := InkMiner{conn: client, address: address}
		neighbours[address] = inkMiner

		return &inkMiner
	}

	return nil
}

/*
	Checks if the current neighbour miner already exists in the list of neighbours
	@param: outgoingAddress of the new neighbour
	@return: true if neighbour outgoingAddress is found; false otherwise
*/
func doesNeighbourExist(addr net.Addr) bool {
	_, exists := neighbours[addr]
	return exists
}

/*
	Checks to see if miner has greater than min number of connections
	@return: Returns true if the miner has enough neighbours
*/
func hasEnoughNeighbours() bool {
	return len(neighbours) >= int(minerNetSettings.MinNumMinerConnections)
}

/*
	Routine for the ink miner to request for more nodes when
	it has less than the min number of miners
	Currently running this routine every second to not kill cpu
	@returns: error when it fails to reach the server
*/
func requestForMoreNodesRoutine() error {
	for range time.Tick(500 * time.Millisecond) {
		if !hasEnoughNeighbours() {
			err := getNodes()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

/*
	Tries to find a nonce such that the hash of the block has the correct
	number of trailing zeros
	Since acquiring a lock can be expensive, this function tries some n nonces
	before giving up the lock, and then trying to re-acquire it
*/
func mine() {
	// number of nonces to try before giving up lock
	numTries := 100

	// so that we don't check the same nonce again and again,
	// keep a value that is always incremented. It will (eventually)
	// roll over, but that's ok; by then, the curBlock will have almost
	// certainly changed
	nonceTry := 0

	// should be trying to mine constantly
	for {
		// acquire lock
		blockLock.Lock()
		for i := 0; i < numTries; i++ {
			currBlock.nonce = strconv.Itoa(nonceTry)
			nonceTry++
			currNonceHash := hashBlock(*currBlock)
			if err := verifyBlockNonce(currNonceHash.ToString(), len(currBlock.ops) == 0); err == nil {
				// currBlock is now a valid block
				// so create BlockMeta to wrap around currBlock
				hash := hashBlock(*currBlock)
				r, s, err := ecdsa.Sign(rand.Reader, &privateKey, hash)
				if err != nil {
					// if encountered an error, just skip this nonce
					continue
				}

				newBlockMeta := &BlockMeta{
					hash:  hash,
					r:     *r,
					s:     *s,
					block: *currBlock,
				}

				// the RPC call does the work we need, so just call it from within this miner
				m := new(MinMin)
				var reply bool
				// if it's successful, we want to start with a clean block
				// if it's unsuccessful, it's the responsability of op routines to add back
				// operations, at which point they will be re-validated (but this case should
				// never actually happen)
				// in both cases, we don't care about the result
				m.NotifyNewBlock(newBlockMeta, &reply)

				block := Block{prev: hash, len: currBlock.len + 1}
				currBlock = &block
				break
			}
		}
		// give up lock
		blockLock.Unlock()

		time.Sleep(time.Second)

		//TODO breaks heartbeat
	}
}

// go run ink-miner.go <serverIP:Port> "`cat <path_to_pub_key>`" "`cat <path_to_priv_key>`"
func main() {
	// ink-miner should take one parameter, which is its outgoingAddress
	// skip program
	args := os.Args[1:]

	numArgs := 3

	// check number of arguments
	if len(args) != numArgs {
		if len(args) < numArgs {
			fmt.Printf("too few arguments; expected %d, received %d\n", numArgs, len(args))
		} else {
			fmt.Printf("too many arguments; expected %d, received %d\n", numArgs, len(args))
		}
		// can't proceed without correct number of arguments
		return
	}

	outgoingAddress = args[0]

	pub, err := hex.DecodeString(args[1])
	if err != nil {
		fmt.Println(args[1])
		panic(err)
	}
	parsedPublicKey, err := x509.ParsePKIXPublicKey(pub)
	if err != nil {
		fmt.Println(err)
		fmt.Printf("miner needs a valid public key")
		return
	}
	priv, _ := hex.DecodeString(args[2])
	parsedPrivateKey, err := x509.ParseECPrivateKey(priv)
	if err != nil {
		fmt.Println(err)
		fmt.Printf("miner needs a valid private key")
	}
	publicKey = *parsedPublicKey.(*ecdsa.PublicKey)
	privateKey = *parsedPrivateKey

	gob.Register(&net.TCPAddr{})
	gob.Register(&elliptic.CurveParams{})
	gob.Register(elliptic.P224())
	gob.Register(elliptic.P256())
	gob.Register(elliptic.P384())
	gob.Register(elliptic.P521())

	client, err := rpc.Dial("tcp", outgoingAddress)
	if err != nil {
		// can't proceed without a connection to the server
		fmt.Printf("miner cannot dial to the server")
		return
	}
	serverConn = client

	// Setup RPC
	server := rpc.NewServer()
	libMin := new(LibMin)
	server.Register(libMin)
	// need automatic port generation
	l, e := net.Listen("tcp", ":0")
	if e != nil {
		fmt.Printf("%v\n", e)
		return
	}
	go server.Accept(l)
	incomingAddress = l.Addr().String()
	// Register miner's incomingAddress
	if registerMinerToServer() != nil {
		// cannot proceed if it is not register to the server
		fmt.Printf("miner cannot register itself to the server")
		return
	}

	//Initializing the first block
	genesisHash, err := hex.DecodeString(minerNetSettings.GenesisBlockHash)
	if err != nil {
		// Only occurs on startup. Panic to prevent miner from running in bad state.
		panic(err)
	}
	hash := blockartlib.Hash(genesisHash)
	currBlock = &Block{prev: hash, len: 1}

	// create genesis block
	genesisBlockMeta := &BlockMeta{hash: hash}
	blockTree[hash.ToString()] = genesisBlockMeta
	headBlockMeta = genesisBlockMeta

	go startHeartBeat()

	go requestForMoreNodesRoutine()

	mine()
}
