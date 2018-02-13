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
	"crypto/md5"
	"crypto/x509"
	"encoding/hex"
	"fmt"
	"net"
	"net/rpc"
	"os"
	"strings"
	"sync"
	"time"
)

// Static
var canvasSettings blockartlib.CanvasSettings
var publicKey ecdsa.PublicKey
var privateKey ecdsa.PrivateKey

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

// Network Instructions
var minerNetSettings rpcCommunication.MinerNetSettings

// FIXME
var ink int // TODO Do we want this? Or do we want a func that scans blockchain before & after op validation

type Op struct {
	shape     *blockartlib.Shape // not nil iff adding shape
	shapeHash string             // non-empty iff removing shape
	owner     string             // hash of pub/priv keys
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

type ServerConnectionError string

func (e ServerConnectionError) Error() string {
	return fmt.Sprintf("InkMiner: Could not connect to server for %s", string(e))
}

type KeyParseError string

func (e KeyParseError) Error() string {
	return fmt.Sprintf("InkMiner: Could not connect to server for %s", string(e))
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

// TODO RPC calls feel a bit burdensome here.
// Receives block flood calls. Verifies chains. Updates head block if new chain is acknowledged.
// LOCKS: Calls headBlockLock()
// @param block *Block: Block which was added to chain.
// @param reply *bool: Bool indicating success of RPC.
// @return error: Any errors produced during new block processing.
func (m *MinMin) NotifyNewBlock(block *Block, reply *bool) error {
	if b := blockTree[hashBlock(*block)]; b != nil {
		// We are already aware of this block.
		return nil
	}

	*reply = false

	// the crawl by default does all the work we need, so no special helper/args/reply is required
	var inter interface{}
	if err := crawlChain(block, nil, inter, inter); err != nil {
		return err
	}

	*reply = true
	headBlockLock.Lock()
	defer headBlockLock.Unlock()

	if block.len > headBlock.len {
		headBlock = block
	}

	floodBlock(*block)

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
// @param reply *blockartlib.ConvasSettings: pointer to CanvasSettings that will be returned
// @return error: Any errors produced
func (l *LibMin) GetCanvasSettings(args int, reply *blockartlib.CanvasSettings) (err error) {
	*reply = canvasSettings
	return nil
}

// Adds a new shape ot the canvas
// @param args *blockartlib.AddShapeArgs: contains the shape to be added, and the validateNum
// @param reply *blockartlib.AddShapeReply: pointer to AddShapeReply that will be returned
// @return err error: Any errors produced
func (l *LibMin) AddShape(args *blockartlib.AddShapeArgs, reply *blockartlib.AddShapeReply) (err error) {
	// construct Op for shape
	op := Op{
		shape:     &args.Shape,
		shapeHash: "",
		owner:     "", // TODO - generate owner hash
	}

	// receiveNewOp will try to add op to current block and flood op
	if err = receiveNewOp(op); err != nil {
		// return error in reply so that it is not cast
		reply.Error = err
	}

	// TODO - wait until args.ValidateNum blocks have been added this block before returning

	return nil
}

// TODO
// Returns the full SvgString fro the given hash, if it exists on the longest
// @param args *blockartlib.GetSvgStringArgs: contains the hash of the shape to be returned
// @param reply *blockartlib.GetSvgStringReply: contains the shape string, and any internal errors
// @param err error: Any errors produced
func (l *LibMin) GetSvgString(args *blockartlib.GetSvgStringArgs, reply *blockartlib.GetSvgStringReply) (err error) {
	// TODO - iterate through headBlock searching for the string
	// For now, just return an InvalidShapeHashError
	reply.Error = blockartlib.InvalidShapeHashError(args.ShapeHash)
	return nil
}

// TODO
// Returns the amount of ink remaining for this miner, in pixels
// @param args args *int: dummy argument that is not used
// @param reply *uint32: amount of remaining ink, in pixels
// @param err error: Any errors produced
func (l *LibMin) GetInk(args *int, reply *uint32) (err error) {
	// TODO - iterate through headBlock to calculate remaining ink
	*reply = 0
	return nil
}

// TODO
// Deletes the shape associated with the passed shapeHash, if it exists and is owned by this miner.
// args.ValidateNum specifies the number of blocks (no-op or op) that must follow the block with this
// operation in the block-chain along the longest path before the operation can return successfully.
// @param args *blockartlib.DeleteShapeArgs: contains the ValidateNum and ShapeHash
// @param reply *blockartlib.DeleteShapeReply: contains the ink remaining, and any internal errors
// @param err error: Any errors produced
func (l *LibMin) DeleteShape(args *blockartlib.DeleteShapeArgs, reply *blockartlib.DeleteShapeReply) (err error) {
	// TODO - iterate through headBlock searching for shape, and delete it
	// Then wait until args.ValidateNum blocks have been added this block before returning
	// For now, just return a ShapeOwnerError
	reply.Error = blockartlib.ShapeOwnerError(args.ShapeHash)
	return nil
}

// TODO
// Returns the shape hashes contained by the block in BlockHash
// @param args *string: the blockHash
// @param reply *blockartlib.GetShapesReply: contains the slice of shape hashes and any internal errors
// @param err error: Any errors produced
func (l *LibMin) GetShapes(args *string, reply *blockartlib.GetShapesReply) (err error) {
	// TODO - search for the block, construct a slice of hashes and return it
	// For now, just return an InvalidBlockHashError
	reply.Error = blockartlib.InvalidBlockHashError(*args)
	return nil
}

// TODO
// Returns the hash of the genesis block of the block chain
// @param args args *int: dummy argument that is not used
// @param reply *uint32: hash of genesis block
// @param err error: Any errors produced
func (l *LibMin) GetGenesisBlock(args *int, reply *string) (err error) {
	// TODO
	*reply = ""
	return nil
}

// TODO
// Returns the shape hashes contained by the block in BlockHash
// @param args *string: the blockHash
// @param reply *blockartlib.GetChildrenReply: contains the slice of block hashes and any internal errors
// @param err error: Any errors produced
func (l *LibMin) GetChildren(args *string, reply *blockartlib.GetChildrenReply) (err error) {
	// TODO - search for children whose parent is the passed BlockHash
	// For now, just return an InvalidBlockHashError
	reply.Error = blockartlib.InvalidBlockHashError(*args)
	return nil
}

// TODO
// Closes the canvas
// @param args args *int: dummy argument that is not used
// @param reply *uint32: hash of genesis block
// @param err error: Any errors produced
func (l *LibMin) CloseCanvas(args *int, reply *string) (err error) {
	// TODO
	*reply = ""
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
//				   @return error: any errors returned by the function
//				   Note that the args and reply must have a struct defintion (like in RPC) and must
//                 be cast to that type in fn, with a call like argsT, ok := args.(Type)
// @return err error: returns any errors encountered, orone of the following errors:
// 		- InvalidBlockHashError
func crawlChain(headBlock *Block, fn func(*Block, interface{}, interface{}) error, args interface{}, reply interface{}) (err error) {
	if fn == nil {
		fn = crawlNoopHelper
	}

	// the chain, starting at headBlock
	chain := []*Block{}
	curr := headBlock
	for {
		// add current element to the end of the chain
		chain = append(chain, curr)
		parent := crawlChainHelperGetBlock(curr.prev)
		if parent == nil {
			// If the parent could not be found, then the hash is invalid.
			return blockartlib.InvalidBlockHashError(hashBlock(*curr))
		}

		if isGenesis(*curr) {
			// We're at the end of the chain.
			break
		}

		curr = parent
	}

	// Validate in reverse order (from GenesisBlock to headBlock).
	for i := len(chain) - 1; i >= 0; i-- {
		block := chain[i]
		hash := hashBlock(*block)
		if _, exists := blockTree[hash]; exists {
			// Block is already stored locally, so has already been validated.
			// Since block has already been validated, all parents of block
			// must also be valid.
			break
		} else {
			// validate block, knowing that all parent blocks are valid
			if err = validateBlock(chain[i:]); err != nil {
				// The block was not valid, return the error.
				return err
			}

			// Block is valid, so add it to the map.
			blockTree[hash] = block
		}
	}

	// Blocks are valid, so now run the function on each block in the chain,
	// starting from the headBlock.
	for i := 0; i < len(chain); i++ {
		if err = fn(chain[i], args, reply); err != nil {
			// if an error is encountered, return it
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
// @return error: always nil
func crawlNoopHelper(block *Block, args interface{}, reply interface{}) error {
	return nil
}

// Returns block with given hash.
// If the block is not stored locally, try to get the block from another miner.
// NOTE: this operation does no verification on any external blocks.
// @param nonce string: The nonce of the block to get info on.
// @return Block: The requested block, or nil if no block is found.
func crawlChainHelperGetBlock(hash string) (block *Block) {
	// Search locally.
	if block = blockTree[hash]; block != nil {
		return block
	}

	// block is not stored locally, search externally
	for _, n := range neighbours {
		err := n.conn.Call("MinMin.RequestBlock", hash, block)
		if err != nil {
			// Block not found, keep searching.
			continue
		}
		// return the block
		return block
	}

	// Block not found.
	return nil
}

// TODO
// Validates the FIRST block in the slice, ASSUMING that all other blocks in the
// chain have already been validated
// @param chain []*Block: the block chain. The first element in the slice is the
//                        block being validated, assume rest of blocks are valid
//                        (and thus the last block should be the Genesis block)
// @return err error: any errors from validation; nil if block is valid
func validateBlock(chain []*Block) (err error) {
	// TODO
	return nil
}

// Returns true if block is the genesis block.
// @param block Block: The block to test against.
// @return bool: True iff block is genesis block.
func isGenesis(block Block) bool {
	// TODO: def'n of Genesis block? ---> Is this the proper hash
	return block.prev == "" && hashBlock(block) == minerNetSettings.GenesisBlockHash
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
	n := int(minerNetSettings.PoWDifficultyOpBlock)
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
		ink, err := blockartlib.InkUsed(op.shape)
		inkAvail := inkAvail(op.owner, headBlock)
		if err != nil || inkAvail < ink {
			// not enough ink
			return blockartlib.InsufficientInkError(inkAvail)
		}
	}

	if op.shape != nil {
		if hash := shapeOverlaps(op.shape, headBlock); hash != "" {
			// op is adding a shape that intersects with an already present shape; reject
			return blockartlib.ShapeOverlapError(hash)
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
// @param shape *blockartlib.Shape: pointer to shape that will be validated
// @return err error: Error indicating if shape is valid. Can be nil or one
//                    of the following errors:
// 						- OutOfBoundsError
func validateShape(shape *blockartlib.Shape) (err error) {
	// TODO
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
// @param shapeHash string: hash of shape to check
// @param owner string: string identfying miner
// @param headBlock *Block: head block of chain from which ink will be calculated
// @return err error: Error indicating if shape is valid. Can be nil or one
//                    of the following errors:
// 						- ShapeOwnerError
//						- TODO - error if shape does not exist?
func shapeExists(shapeHash string, ownder string, headBlock *Block) (err error) {
	// TODO
	return blockartlib.ShapeOwnerError(shapeHash)
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

/*
	Registering the miner to the server, calling the server's RPC method
	@return error: ServerConnectionError if connection to server fails
*/
func registerMinerToServer() error {
	tcpAddr, err := net.ResolveTCPAddr("tcp", address)
	if err != nil {
		return ServerConnectionError("resolve tcp error")
	}
	minerSettings := rpcCommunication.MinerInfo{Address: tcpAddr, Key: publicKey}
	clientErr := serverConn.Call("RServer.Register", &minerSettings, &minerNetSettings)
	if clientErr != nil {
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
		clientErr := serverConn.Call("RServer.HeartBeat", &publicKey, &reply)
		if clientErr != nil {
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
	var neighbourAddresses *[]net.Addr
	clientErr := serverConn.Call("RServer.GetNodes", &publicKey, &neighbourAddresses)
	if clientErr != nil {
		return ServerConnectionError("get nodes failure")
	}

	neighboursLock.Lock()
	for _, address := range *neighbourAddresses {
		inkMiner := InkMiner{}
		client, err := rpc.Dial(address.Network(), address.String())
		if err != nil {
			// if we can not connect to a node, just try the next one
			continue
		}
		inkMiner.conn = client
		neighbours = append(neighbours, &inkMiner)
	}
	neighboursLock.Unlock()

	return nil
}

func hasEnoughNeighbours() bool {
	hasEnough := false
	neighboursLock.Lock()
	if len(neighbours) >= int(minerNetSettings.MinNumMinerConnections) {
		hasEnough = true
	}
	neighboursLock.Unlock()
	return hasEnough
}

func main() {
	// ink-miner should take one parameter, which is its address
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

	address = args[0]

	//TODO: verify if this parse is this correct?
	parsedPublicKey, err := x509.ParsePKIXPublicKey([]byte(args[1]))
	if err != nil {
		// can't proceed without a proper public key
		fmt.Printf("miner needs a valid public key")
		return
	}

	parsedPrivateKey, err := x509.ParseECPrivateKey([]byte(args[2]))
	if err != nil {
		// can't proceed without a proper private key
		fmt.Printf("miner needs a valid private key")
		return
	}

	publicKey = parsedPublicKey.(ecdsa.PublicKey)
	privateKey = *parsedPrivateKey

	// TODO - should communicate with server to get CanvasSettings and other miners in the network
	client, err := rpc.Dial("tcp", address)
	if err != nil {
		// can't proceed without a connection to the server
		fmt.Printf("miner can not dial to the server")
		return
	}
	serverConn = client
	if registerMinerToServer() != nil {
		// can not proceed if it is not register to the server
		fmt.Printf("miner can not register itself to the server")
		return
	}
	go startHeartBeat()

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
