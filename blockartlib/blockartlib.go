/*

This package specifies the application's interface to the the BlockArt
library (blockartlib) to be used in project 1 of UBC CS 416 2017W2.

*/

package blockartlib

import "crypto/ecdsa"
import (
	"fmt"
	"sync"
	"net/rpc"
)

// Represents a type of shape in the BlockArt system.
type ShapeType int

const (
	// Path shape.
	PATH ShapeType = iota

	// Circle shape (extra credit).
	// CIRCLE
)

// Settings for a canvas in BlockArt.
type CanvasSettings struct {
	// Canvas dimensions
	CanvasXMax uint32
	CanvasYMax uint32
}

// Settings for an instance of the BlockArt project/network.
type MinerNetSettings struct {
	// Hash of the very first (empty) block in the chain.
	GenesisBlockHash string

	// The minimum number of ink miners that an ink miner should be
	// connected to. If the ink miner dips below this number, then
	// they have to retrieve more nodes from the server using
	// GetNodes().
	MinNumMinerConnections uint8

	// Mining ink reward per op and no-op blocks (>= 1)
	InkPerOpBlock   uint32
	InkPerNoOpBlock uint32

	// Number of milliseconds between heartbeat messages to the server.
	HeartBeat uint32

	// Proof of work difficulty: number of zeroes in prefix (>=0)
	PoWDifficultyOpBlock   uint8
	PoWDifficultyNoOpBlock uint8

	// Canvas settings
	canvasSettings CanvasSettings
}

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

////////////////////////////////////////////////////////////////////////////////////////////
// <ERROR DEFINITIONS>

// These type definitions allow the application to explicitly check
// for the kind of error that occurred. Each API call below lists the
// errors that it is allowed to raise.
//
// Also see:
// https://blog.golang.org/error-handling-and-go
// https://blog.golang.org/errors-are-values

// Contains address IP:port that art node cannot connect to.
type DisconnectedError string

func (e DisconnectedError) Error() string {
	return fmt.Sprintf("BlockArt: cannot connect to [%s]", string(e))
}

// Contains amount of ink remaining.
type InsufficientInkError uint32

func (e InsufficientInkError) Error() string {
	return fmt.Sprintf("BlockArt: Not enough ink to addShape [%d]", uint32(e))
}

// Contains the offending svg string.
type InvalidShapeSvgStringError string

func (e InvalidShapeSvgStringError) Error() string {
	return fmt.Sprintf("BlockArt: Bad shape svg string [%s]", string(e))
}

// Contains the offending svg string.
type ShapeSvgStringTooLongError string

func (e ShapeSvgStringTooLongError) Error() string {
	return fmt.Sprintf("BlockArt: Shape svg string too long [%s]", string(e))
}

// Contains the bad shape hash string.
type InvalidShapeHashError string

func (e InvalidShapeHashError) Error() string {
	return fmt.Sprintf("BlockArt: Invalid shape hash [%s]", string(e))
}

// Contains the bad shape hash string.
type ShapeOwnerError string

func (e ShapeOwnerError) Error() string {
	return fmt.Sprintf("BlockArt: Shape owned by someone else [%s]", string(e))
}

// Empty
type OutOfBoundsError struct{}

func (e OutOfBoundsError) Error() string {
	return fmt.Sprintf("BlockArt: Shape is outside the bounds of the canvas")
}

// Contains the hash of the shape that this shape overlaps with.
type ShapeOverlapError string

func (e ShapeOverlapError) Error() string {
	return fmt.Sprintf("BlockArt: Shape overlaps with a previously added shape [%s]", string(e))
}

// Contains the invalid block hash.
type InvalidBlockHashError string

func (e InvalidBlockHashError) Error() string {
	return fmt.Sprintf("BlockArt: Invalid block hash [%s]", string(e))
}

// </ERROR DEFINITIONS>
////////////////////////////////////////////////////////////////////////////////////////////

// Represents a canvas in the system.
type Canvas interface {
	// Adds a new shape to the canvas.
	// Can return the following errors:
	// - DisconnectedError
	// - InsufficientInkError
	// - InvalidShapeSvgStringError
	// - ShapeSvgStringTooLongError
	// - ShapeOverlapError
	// - OutOfBoundsError
	AddShape(validateNum uint8, shapeType ShapeType, shapeSvgString string, fill string, stroke string) (shapeHash string, blockHash string, inkRemaining uint32, err error)

	// Returns the encoding of the shape as an svg string.
	// Can return the following errors:
	// - DisconnectedError
	// - InvalidShapeHashError
	GetSvgString(shapeHash string) (svgString string, err error)

	// Returns the amount of ink currently available.
	// Can return the following errors:
	// - DisconnectedError
	GetInk() (inkRemaining uint32, err error)

	// Removes a shape from the canvas.
	// Can return the following errors:
	// - DisconnectedError
	// - ShapeOwnerError
	DeleteShape(validateNum uint8, shapeHash string) (inkRemaining uint32, err error)

	// Retrieves hashes contained by a specific block.
	// Can return the following errors:
	// - DisconnectedError
	// - InvalidBlockHashError
	GetShapes(blockHash string) (shapeHashes []string, err error)

	// Returns the block hash of the genesis block.
	// Can return the following errors:
	// - DisconnectedError
	GetGenesisBlock() (blockHash string, err error)

	// Retrieves the children blocks of the block identified by blockHash.
	// Can return the following errors:
	// - DisconnectedError
	// - InvalidBlockHashError
	GetChildren(blockHash string) (blockHashes []string, err error)

	// Closes the canvas/connection to the BlockArt network.
	// - DisconnectedError
	CloseCanvas() (inkRemaining uint32, err error)
}

// Canvas type that implements the Canvas interface
type CanvasT struct {
	minerAddr string
	privKey ecdsa.PrivateKey
	client *rpc.Client
}

func (canvas *CanvasT) AddShape(validateNum uint8, shapeType ShapeType, shapeSvgString string, fill string, stroke string) (shapeHash string, blockHash string, inkRemaining uint32, err error) {
	// convert parameters to internal shape representation
	shape, e := convertShape(shapeType, shapeSvgString, fill, stroke)
	if e != nil {
		// TODO - deal with any errors convertShape may produce
		return shapeHash, blockHash, inkRemaining, e
	}

	args := &AddShapeArgs{
		Shape: shape,
		ValidateNum: validateNum}
	var reply AddShapeReply
	e = canvas.client.Call("LimMin.AddShape", args, &reply)
	if reply.Error != nil {
		return shapeHash, blockHash, inkRemaining, reply.Error
	} else if e != nil {
		return shapeHash, blockHash, inkRemaining, DisconnectedError(canvas.minerAddr)
	}
	
	return reply.ShapeHash, reply.BlockHash, reply.InkRemaining, nil
}

// TODO - merge conflict @Wesley @Justin
// 	      I assume you've already written this function; please delete it and fix places that call this function
// Can return the following errors:
// - InvalidShapeSvgStringError
// - ShapeSvgStringTooLongError
func convertShape(shapeType ShapeType, shapeSvgString string, fill string, stroke string) (shape Shape, err error) {
	// TODO
	return shape, nil
}

func (canvas *CanvasT) GetSvgString(shapeHash string) (svgString string, err error) {
	// TODO
	return "", nil
}

func (canvas *CanvasT) GetInk() (inkRemaining uint32, err error) {
	// TODO
	return 0, nil
}

func (canvas *CanvasT) DeleteShape(validateNum uint8, shapeHash string) (inkRemaining uint32, err error) {
	// TODO
	return 0, nil
}

func (canvas *CanvasT) GetShapes(blockHash string) (shapeHashes []string, err error) {
	// TODO
	return nil, nil
}

func (canvas *CanvasT) GetGenesisBlock() (blockHash string, err error) {
	// TODO
	return "", nil
}

func (canvas *CanvasT) GetChildren(blockHash string) (blockHashes []string, err error) {
	// TODO
	return nil, nil
}

func (canvas *CanvasT) CloseCanvas() (inkRemaining uint32, err error) {
	// TODO
	return 0, nil
}

// OpenCanvas returns a singleton Canvas
// Idea for singleton implementation based off https://stackoverflow.com/questions/1823286/singleton-in-go
var canvasT *CanvasT
var once sync.Once

// The constructor for a new Canvas object instance. Takes the miner's
// IP:port address string and a public-private key pair (ecdsa private
// key type contains the public key). Returns a Canvas instance that
// can be used for all future interactions with blockartlib.
//
// The returned Canvas instance is a singleton: an application is
// expected to interact with just one Canvas instance at a time.
//
// Can return the following errors:
// - DisconnectedError
func OpenCanvas(minerAddr string, privKey ecdsa.PrivateKey) (canvas Canvas, setting CanvasSettings, err error) {
	// make thread-safe singleton intialization
	once.Do(func() {
		canvasT = &CanvasT{}
	})

	canvasT.minerAddr = minerAddr
	canvasT.privKey = privKey

	// connect to miner
	if canvasT.client, err = rpc.Dial("tcp", minerAddr); err != nil {
		return canvasT, setting, DisconnectedError(minerAddr)
	}

	if err = canvasT.client.Call("LimMin.GetCanvasSettings", 0, &setting); err != nil {
		return canvasT, setting, DisconnectedError(minerAddr)
	}

	return canvasT, setting, nil
}

// TODO
// - calculates the amount of ink required to draw the shape, in pixels
// @param shape *blockartlib.Shape: pointer to shape whose ink cost will be calculated
// @return ink int: amount of ink required to draw the shape
// @return error err: TODO
func InkUsed(shape *Shape) (ink int, err error) {
	ink = 0
	// get border length of shape

	return ink, nil
}

