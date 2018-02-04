/*

This package specifies the application's interface to the the BlockArt
library (blockartlib) to be used in project 1 of UBC CS 416 2017W2.

*/

package blockartlib

import "crypto/ecdsa"
import (
	"fmt"
	"regexp"
	"strconv"
	"unicode"
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

type Edge struct {
	startPoint , endPoint Point
}

type Shape struct {
	hash string
	svg string
	edges []Edge
	filledIn bool
	ink int
	closedWithZ bool
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
	// TODO
	// For now return DisconnectedError
	return nil, CanvasSettings{}, DisconnectedError("")
}

func checkErr(err error){
	if err != nil {
		panic(err)
	}
}

func checkSvgStringlen(svgString string) bool{
	return len(svgString) > 128
}


/* TODO: a lot of edge cases here for the svg
	- lifting up the pen nultiple "m"
	- different key words , filter out id="" or something with two d's
*/

func SvgToShape(svgString string) (Shape, error) {
	if checkSvgStringlen(svgString){
		return Shape{}, ShapeSvgStringTooLongError("Svg string has too many characters")
	}
	re, err  := regexp.Compile(" d=\".*\"\\/>")
	checkErr(err)
	matches := re.FindAllString(svgString , -1)
	if matches != nil {
		//TODO for-loop it ? can have multiple paths?
		isFilledIn := checkIsFilled(matches[0])
		path := getDPoints(matches[0])
		shape, err := parseSvgPath(path)
		if err != nil {
			return Shape{}, err
		}
		shape.svg = svgString
		shape.filledIn = isFilledIn
		fmt.Println(shape)
		return shape , nil
	}
	return Shape{}, InvalidShapeSvgStringError("not a valid shape")
}

func checkIsFilled(path string) bool {
	re , err := regexp.Compile("fill=\".*\"")
	checkErr(err)
	// checking for fill
	matches := re.FindAllString(path, -1)
	if matches != nil {
		isTransparent , err := regexp.MatchString("\"transparent\"", matches[0])
		checkErr(err)
		return !isTransparent
	}
	return false
}

func getDPoints(svgPath string) string {
	re, err  := regexp.Compile("d=\".*?\"")
	checkErr(err)
	matches := re.FindAllString(svgPath , 1)
	if matches != nil {
		return matches[0]
	}
	return ""
}

func parseSvgPath(path string) (Shape, error) {
	fmt.Println(path)
	shape := Shape{}
	currXPoint := 0
	currYPoint := 0
	index := 0

	originXPoint := 0
	originYPoint := 0

	for {
		if index >= len(path) {
			break
		}

		char := path[index]
		s := string(char)
		fmt.Println("Position " + strconv.Itoa(index) + " looking at this char '" + s + "'")

		if s == "M" || s == "m"{
			xPoint := 0
			yPoint := 0
			onFirstNum := false
			onSecondNum := false
			finishedFirstNumber := false
			finishedSecondNumber := false

			for i := index + 1; i < len(path); i++ {
				letter := path[i]
				rLetter := rune(letter)
				if unicode.IsNumber(rLetter) {
					if onFirstNum == false {
						onFirstNum = true
					}
					if onSecondNum == false {
						onSecondNum = true
					}

					if finishedSecondNumber == true {
						fmt.Println("errored")
						return Shape{}, InvalidShapeSvgStringError("can not have more than three numbers behind M")
					}

					num, err := strconv.Atoi(string(rLetter))
					checkErr(err)
					if !finishedFirstNumber {
						xPoint = xPoint*10 + num
					} else {
						yPoint = yPoint*10 + num
					}
				}
				if unicode.IsSpace(rLetter) {
					if finishedFirstNumber == false && onFirstNum == true {
						// finished first number
						finishedFirstNumber = true
					} else if finishedSecondNumber == false && onSecondNum == true {
						finishedSecondNumber = true
					}
				}
				if !unicode.IsNumber(rLetter) && !unicode.IsSpace(rLetter) {
					break
				}
				index++
			}

			if s == "M" {
				currXPoint = xPoint
				currYPoint = yPoint
			} else {
				currXPoint = currXPoint + xPoint
				currYPoint = currYPoint + yPoint
			}

			originXPoint = currXPoint
			originYPoint = currYPoint
		}

		if s == "H" || s == "V" || s == "h" || s == "v" {
			//getting the value
			value := 0
			for i := index + 1; i < len(path); i++ {
				letter := path[i]
				rLetter := rune(letter)
				if unicode.IsNumber(rLetter) {
					num, err := strconv.Atoi(string(rLetter))
					checkErr(err)
					value = value*10 + num
				}
				if !unicode.IsNumber(rLetter) && !unicode.IsSpace(rLetter) {
					break
				}
				index++
			}
			// assigning it
			if s == "H" || s == "h" {
				startPoint := Point{currXPoint, currYPoint}
				var endPoint Point
				endPoint.y = currYPoint
				if s == "H"{
					endPoint.x = value
					currXPoint = value
				} else {
					endPoint.x = currXPoint + value
					currXPoint = currXPoint + value
				}
				edge := Edge{startPoint, endPoint}
				shape.edges = append(shape.edges, edge)

			}
			if s == "V" || s == "v" {
				startPoint := Point{currXPoint, currYPoint}
				var endPoint Point
				endPoint.x = currXPoint
				if s == "V"{
					endPoint.y = value
					currYPoint = value
				} else {
					endPoint.y = currYPoint + value
					currYPoint = currYPoint + value
				}
				edge := Edge{startPoint, endPoint}
				shape.edges = append(shape.edges, edge)
			}
		}

		if s == "L" || s == "l" {
			xPoint := 0
			yPoint := 0
			onFirstNum := false
			onSecondNum := false
			finishedFirstNumber := false
			finishedSecondNumber := false
			for i := index + 1; i < len(path); i++ {
				letter := path[i]
				rLetter := rune(letter)
				if unicode.IsNumber(rLetter) {
					if onFirstNum == false {
						onFirstNum = true
					}
					if onSecondNum == false {
						onSecondNum = true
					}

					if finishedSecondNumber == true {
						fmt.Println("errored")
						return Shape{}, InvalidShapeSvgStringError("can not have more than three numbers behind L")
					}

					num, err := strconv.Atoi(string(rLetter))
					checkErr(err)
					if !finishedFirstNumber {
						xPoint = xPoint*10 + num
					} else {
						yPoint = yPoint*10 + num
					}
				}
				if unicode.IsSpace(rLetter) {
					if finishedFirstNumber == false && onFirstNum == true {
						// finished first number
						finishedFirstNumber = true
					} else if finishedSecondNumber == false && onSecondNum == true {
						finishedSecondNumber = true
					}
				}
				if !unicode.IsNumber(rLetter) && !unicode.IsSpace(rLetter) {
					break
				}
				index++
			}

			fmt.Println(xPoint , yPoint)
			startPoint := Point{currXPoint, currYPoint}
			var endPoint Point
			if s == "L" {
				endPoint.x = xPoint
				endPoint.y = yPoint
			} else {
				endPoint.x = currXPoint + xPoint
				endPoint.y = currYPoint + yPoint
			}
			currXPoint = endPoint.x
			currYPoint = endPoint.y
			edge := Edge{startPoint, endPoint}
			shape.edges = append(shape.edges, edge)
		}

		if s == "Z" || s == "z" {
			edge := Edge{startPoint:Point{currXPoint, currYPoint}, endPoint:Point{originXPoint, originYPoint}}
			shape.edges = append(shape.edges, edge)
			shape.closedWithZ = true
		}
		index++
	}
	
	return shape , nil
}

func InkUsed(shape *Shape) (ink int, err error) {
	ink = 0
	// get border length of shape

	return ink, nil
}