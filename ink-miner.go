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
	"encoding/hex"
	"crypto/x509"
	"encoding/pem"
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
var blockTree map[string]*BlockMeta
var serverConn *rpc.Client
var outgoingAddress string
var incomingAddress string

// Network Instructions
var minerNetSettings *rpcCommunication.MinerNetSettings

// slice of operation threads' channels that need to know about new blocks
var opChans = make(map[string](chan *BlockMeta))
var opChansLock = &sync.Mutex{}

// FIXME
var ink int // TODO Do we want this? Or do we want a func that scans blockchain before & after op validation

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

func (o Op) String() string {
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

func (b Block) String() string {
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
// @return error: TODO
func (m *MinMin) NotifyNewOp(opMeta *OpMeta, reply *bool) (err error) {
	// TODO - check if op has already been seen, and only flood if it is new
	// if op is validated, receiveNewOp will put op in currBlock and flood the op
	*reply = false
	if e := receiveNewOp(*opMeta); e == nil {
		// validate was successful only if error is null
		// TODO - is the error  useful?
		*reply = true
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
		newOps := []OpMeta{}
		verificationChan := make(chan error, 1)
		for _, oldOp := range currBlock.ops {
			// go through ops sequentially for simplicity
			// TODO - if runtime is really bad, could make it parallel
			go verifyOp(oldOp, blockMeta, verificationChan)
			err := <-verificationChan
			if err == nil {
				// op is still valid (wasn't added in a previous block)
				newOps = append(newOps, oldOp)
			}
		}
		close(verificationChan)

		currBlock.ops = newOps
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
	// TODO: Do we want canvas settings?
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
	}
	return nil
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
	shape := findShape(args.ShapeHash)
	if shape != nil {
		// shape does not exist, return InvalidShapeHashError
		reply.Error = blockartlib.InvalidShapeHashError(args.ShapeHash)
		return nil
	}

	// Return html-valid tag, of the form:
	// <path d=[svgString] stroke=[stroke] fill=[fill]/>
	reply.SvgString = fmt.Sprintf("<path d=\"%s\" stroke=\"%s\" fill=\"%s\"/>", shape.Svg, shape.BorderColor, shape.FillColor)
	reply.Error = nil
	return nil
}

// Returns the amount of ink remaining for this miner, in pixels
// @param args args *int: dummy argument that is not used
// @param reply *uint32: amount of remaining ink, in pixels
// @param err error: Any errors produced
func (l *LibMin) GetInk(args *blockartlib.GetInkArgs, reply *uint32) (err error) {
	// acquire currBlock's lock
	// TODO - is this needed? it's read-only (is it?)
	blockLock.Lock()
	defer blockLock.Unlock()

	*reply = inkAvail(args.Miner, currBlock)
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
	if minerNetSettings.GenesisBlockHash == blockartlib.Hash([]byte{}).String() {
		return GensisBlockNotFound("")
	}
	*reply, _ = hex.DecodeString(minerNetSettings.GenesisBlockHash)
	return nil
}

// TODO
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
			if cur, ok = blockTree[cur.block.prev.String()]; !ok {
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
	return opMeta1.hash.String() == opMeta2.hash.String() && opMeta1.r.Cmp(&opMeta2.r) == 0 && opMeta1.s.Cmp(&opMeta2.s) == 0 && opMeta1.op == opMeta2.op
}

// Compares two blockMetas, and returns true if they are equal
// @param block1: the first blockMeta to compare
// @param block2: the second blockMeta to compare
// @return bool: true if the blockMetas are equal, false otherwise
func blockMetasEqual(blockMeta1 BlockMeta, blockMeta2 BlockMeta) bool {
	if blockMeta1.hash.String() != blockMeta2.hash.String() || blockMeta1.r.Cmp(&blockMeta2.r) != 0 || blockMeta1.s.Cmp(&blockMeta2.s) != 0 {
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
	if block1.prev.String() != block2.prev.String() || block1.len != block2.len || block1.nonce != block2.nonce || len(block1.ops) != len(block2.ops) {
		return false
	}

	for i := 0; i < len(block1.ops); i++ {
		if !opMetasEqual(block1.ops[i], block2.ops[i]) {
			return false
		}
	}

	return true
}

// Searches for a shape in the set of local blocks with the given hash.
// Note: this function does not care whether the shape was later deleted
// @param deleteShapeHash string: hash of shape that is being searched for
// @return shape *blockartlib.Shape: found shape whose hash matches deleteShapeHash; nil if it does not
//                                   exist or was deleted
func findShape(deleteShapeHash string) (shape *blockartlib.Shape) {
	// Iterate through all locally stored blocks to search for a shape with the passed hash
	for _, blockMeta := range blockTree {
		block := blockMeta.block
		// search through the block, searching for the add op for a shape with this hash
		for _, opMeta := range block.ops {
			op := opMeta.op
			if op.shapeMeta != nil && op.shapeMeta.Hash == deleteShapeHash {
				// shape was found
				return &op.shapeMeta.Shape
				// keep searching through the block in case it is later deleted
			}
		}
	}

	// shape was not found
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
	if blockMeta, ok := blockTree[hash.String()]; ok && blockMeta != nil {
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
	if hashBlock(blockMeta.block).String() != blockMeta.hash.String() {
		return blockartlib.InvalidBlockHashError(blockMeta.hash)
	}
	// Verify block signature.
	if !ecdsa.Verify(&blockMeta.block.miner, blockMeta.hash, &blockMeta.r, &blockMeta.s) {
		return blockartlib.InvalidBlockHashError(string(blockMeta.hash))
	}
	// Verify POW.
	if err = verifyBlockNonce(blockMeta.block.nonce); err != nil {
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
	// TODO: What is gensis def'n? Who signs it?
	// TODO: def'n of Genesis block? ---> Is this the proper hash
	return string(block.prev) == "" && hashBlock(block).String() == minerNetSettings.GenesisBlockHash
}

// TODO: Might not be worth doing, but do we need seperate hash functions?

// Returns hash of block.
// @param block Block: Block to be hashed.
// @return Hash: The hash of the block.
func hashBlock(block Block) blockartlib.Hash {
	hasher := md5.New()
	hasher.Write([]byte(block.String()))
	return blockartlib.Hash(hasher.Sum(nil)[:])
}

// Returns hash of op.
// @param op Op: Op to be hashed.
// @return Hash: The hash of the op.
func hashOp(op Op) blockartlib.Hash {
	hasher := md5.New()
	hasher.Write([]byte(op.String()))
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
func verifyBlockNonce(hash string) error {
	// TODO - if PoWDifficultyNoOpBlock is different, need to check if there are any ops
	n := int(minerNetSettings.PoWDifficultyOpBlock)
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
	for i, opMeta := range block.ops {
		op := opMeta.op
		if err := validateShape(op.shapeMeta); err != nil {
			return err
		}

		// Ensure op does not conflict with previous ops.
		for j := 0; j < i; j++ {
			if jOp := block.ops[j].op; op.owner != jOp.owner {
				if blockartlib.ShapesIntersect(op.shapeMeta.Shape, jOp.shapeMeta.Shape, minerNetSettings.CanvasSettings) {
					return blockartlib.ShapeOverlapError(string(op.shapeMeta.Hash))
				}
			}
		}
		go verifyOp(opMeta, blockTree[block.prev.String()], verificationChan)
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

// Verifies an op against all previous ops in the blockchain. Assumes all previous blocks in chain are valid.
// LOCKS: Acquires headBlockLock.
// @param candidateOp Op: The op to verify.
// @param block *Block: The headBlock on which to begin the verification.
// @param ch chan<-error: The channel to which verification errors are piped into, or nil if no errors are found.
// FIXME: Must work with currBlock which is not a BlockMeta.
func verifyOp(candidateOpMeta OpMeta, blockMeta *BlockMeta, ch chan<- error) {
	headBlockLock.Lock()
	defer headBlockLock.Unlock()

	// Verify hash.
	if hashOp(candidateOpMeta.op).String() != candidateOpMeta.hash.String() {
		ch <- blockartlib.InvalidShapeHashError(candidateOpMeta.hash.String())
		return
	}

	candidateOp := candidateOpMeta.op

	// Verify signature.
	if !ecdsa.Verify(&candidateOp.owner, candidateOpMeta.hash, &candidateOpMeta.r, &candidateOpMeta.s) {
		ch <- blockartlib.InvalidShapeHashError(candidateOpMeta.hash.String())
		return
	}

	// Verify op with shape.
	if candidateOp.shapeMeta != nil {
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
		ink, err := blockartlib.InkUsed(&shape)
		inkAvail := inkAvail(candidateOp.owner, &headBlockMeta.block)
		if err != nil || inkAvail < ink {
			ch <- blockartlib.InsufficientInkError(inkAvail)
			return
		}

		// Ensure op is not duplicate and shape does not overlap with other ops in the chain.
		curr := blockMeta
		for {
			for _, opMeta := range curr.block.ops {
				// This op has been performed before.
				if candidateOpMeta.hash.String() == opMeta.hash.String() {
					// TODO: More specific error?
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
			curr, ok = blockTree[curr.block.prev.String()]
			if !ok {
				ch <- blockartlib.InvalidBlockHashError(curr.block.prev.String())
				return
			}
		}
		// Verify deletion op.
	} else {
		if err := shapeExists(candidateOp.deleteShapeHash, candidateOp.owner, blockMeta); err != nil {
			ch <- err
			return
		}
	}
	ch <- nil
	return
}

// TODO this and floodBlock currentl share almost all the code. If worth it, call helper
//      function that takes the function and paramters.
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
	// FIXME: verifyOp needs to be fixed. 2nd arg should be currBlock!!!
	deleteMePleaseCurrBlockMeta := BlockMeta{block: *currBlock}
	verifyOp(opMeta, &deleteMePleaseCurrBlockMeta, verifyCh)

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
		// TODO: No custom error type?
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

	// TODO: Check if shape is self-intersecting. Depends on Justin's PR.
	//       Invoke function `isSimpleShape`

	return blockartlib.OutOfBoundsError{}
}

// TODO
// - checks if the passed shape intersects with any shape currently on the canvas
//   that is NOT owned by this miner, starting at headBlock
// - ASSUMES that if any locks are requred for headBlock, they have already been acquired
// @param shape *blockartlib.Shape: pointer to shape that will be checked for
//                                  intersections
// @param headBlock *Block: head block of chain from which ink will be calculated
// @return shapeOverlapHash string: empty if shape does intersect with any other
//                                  non-owned shape; otherwise it is the hash of
//                                  the shape this shape overlaps
func shapeOverlaps(shape *blockartlib.Shape, headBlock *Block) (shapeOverlapHash string) {
	// TODO
	return ""
}

// TODO
// Checks if a shape with the given hash exists on the canvas (and was not later
// deleted) starting at headBlock, and that the passed owner is the owner of this shape
// Returned error is nil if shape does exist and is owned by owner, otherwise returns
// a non-nil error
// - ASSUMES that if any locks are requred for headBlock, they have already been acquired
// @param deleteShapeHash string: hash of shape to check
// @param owner ecdsa.PublicKey: public key identfying miner
// @param headBlock *Block: head block of chain from which ink will be calculated
// @return err error: Error indicating if shape is valid. Can be nil or one
//                    of the following errors:
// 						- ShapeOwnerError
//						- TODO - error if shape does not exist?
func shapeExists(deleteShapeHash string, owner ecdsa.PublicKey, headBlock *BlockMeta) (err error) {
	// TODO
	return blockartlib.ShapeOwnerError(deleteShapeHash)
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
// @return ink uint32: ink currently available to this miner, in pixels
func inkAvail(miner ecdsa.PublicKey, headBlock *Block) (ink uint32) {
	// the crawl by default does all the work we need, so no special helper/args/reply is required
	args := &inkAvailCrawlArgs{miner: miner}
	var reply inkAvailCrawlReply

	// FIXME: crawlChain operates on published chain, not unpublished currBlock.
	deleteMeHeadBlockMeta := BlockMeta{block: *headBlock}
	if err := crawlChain(&deleteMeHeadBlockMeta, inkAvailCrawlHelper, args, &reply); err != nil {
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
	clientErr := serverConn.Call("RServer.Register", &minerSettings, &minerNetSettings)
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
	for range time.Tick(time.Millisecond * time.Duration(minerNetSettings.HeartBeat) / 2) {
		var reply bool
		// passing the miners public key and a dummy reply
		clientErr := serverConn.Call("RServer.HeartBeat", &publicKey, &reply)
		if clientErr != nil {
			//TODO: what do we want to do if the rpc call fails
			return ServerConnectionError("heartbeat failure")
		}
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
		// only add it if the neighbor does not already exist
		if !doesNeighbourExist(address) {
			client, err := rpc.Dial(address.Network(), address.String())
			if err != nil {
				// if we can not connect to a node, just try the next outgoingAddress
				continue
			} else {
				inkMiner := InkMiner{}
				inkMiner.conn = client
				inkMiner.address = address
				neighbours[address] = inkMiner
			}
		}
	}

	neighboursLock.Unlock()
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
	numTries := 1000

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
			if err := verifyBlockNonce(currNonceHash.String()); err == nil {
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
	}
}

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

	// TODO: Uncomment the below:

	//TODO: verify if this parse is this correct?
	pub, _ := pem.Decode([]byte(args[1]))
	parsedPublicKey, err := x509.ParsePKIXPublicKey(pub.Bytes)
	if err != nil {
		fmt.Println(err)
		// can't proceed without a proper public key
		fmt.Printf("miner needs a valid public key")
		return
	}
	priv, _ := pem.Decode([]byte(args[2]))
	parsedPrivateKey, err := x509.ParsePKCS1PrivateKey(priv.Bytes)
	if err != nil {
		// can't proceed without a proper private key
		fmt.Println(err)
		fmt.Printf("miner needs a valid private key")
		return
	}

	publicKey = parsedPublicKey.(ecdsa.PublicKey)
	privateKey = *parsedPrivateKey

	//keyPointer, _ := ecdsa.GenerateKey(elliptic.P224(), rand.Reader)
	//privKey := *keyPointer
	//publicKey = privKey.PublicKey
	//privateKey = privKey

	// TODO -> so we should not need to use P224 or 226 in our encryption
	gob.Register(&net.TCPAddr{})
	gob.Register(&elliptic.CurveParams{})
	gob.Register(elliptic.P224())
	gob.Register(elliptic.P256())
	gob.Register(elliptic.P384())
	gob.Register(elliptic.P521())

	client, err := rpc.Dial("tcp", outgoingAddress)
	if err != nil {
		// can't proceed without a connection to the server
		fmt.Printf("miner can not dial to the server")
		return
	}
	serverConn = client

	// Setup RPC
	server := rpc.NewServer()
	libMin := new(LibMin)
	server.Register(libMin)
	// need automatic port generation
	ip := strings.Split(outgoingAddress, ":")
	l, e := net.Listen("tcp", ip[0]+":0")
	if e != nil {
		fmt.Printf("%v\n", e)
		return
	}
	go server.Accept(l)
	incomingAddress = l.Addr().String()
	// Register miner's incomingAddress
	if registerMinerToServer() != nil {
		// can not proceed if it is not register to the server
		fmt.Printf("miner can not register itself to the server")
		return
	}

	go startHeartBeat()

	go requestForMoreNodesRoutine()

	go mine()

	for {

	}
}
