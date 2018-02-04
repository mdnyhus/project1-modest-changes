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
	"math"
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
		shape := parseSvgPath(path)
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

func parseSvgPath(path string) Shape {
	fmt.Println(path)
	shape := Shape{}
	currXPoint := 0
	currYPoint := 0
	index := 0

	for {
		if index >= len(path) {
			break
		}

		char := path[index]
		s := string(char)
		fmt.Println("Position " + strconv.Itoa(index) + " looking at this char '" + s + "'")

		if s == "M" {
			onFirstNum := false
			foundFirstNum := false
			for i := index + 1; i < len(path); i++ {
				letter := path[i]
				rLetter := rune(letter)
				if unicode.IsNumber(rLetter) {
					if onFirstNum == false {
						onFirstNum = true
					}
					num, err := strconv.Atoi(string(rLetter))
					checkErr(err)
					if !foundFirstNum {
						currXPoint = currXPoint*10 + num
					} else {
						currYPoint = currYPoint*10 + num
					}
				}
				if unicode.IsSpace(rLetter) {
					if foundFirstNum == false && onFirstNum == true {
						foundFirstNum = true
					}
				}
				if !unicode.IsNumber(rLetter) && !unicode.IsSpace(rLetter) {
					break
				}
				index++
			}
			fmt.Println(currXPoint)
			fmt.Println(currYPoint)
			shape.point = append(shape.point, Point{x: currXPoint, y: currYPoint})
		}

		if s == "m" {
			// do we support multiple draws
		}

		if s == "H" || s == "V" {
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
			fmt.Println("Value for " + s + " is: " + strconv.Itoa(value))
			if s == "H" {
				shape.point = append(shape.point, Point{x: value, y: currYPoint})
				currXPoint = value
			}
			if s == "V" {
				shape.point = append(shape.point, Point{x: currXPoint, y: value})
			}
			currYPoint = value
		}

		if s == "h" || s == "v" {

		}

		if s == "L" || s == "l" {
			foundFirstNum := false
			for i := index + 1; i < len(path); i++ {
				letter := path[i]
				rLetter := rune(letter)
				if unicode.IsNumber(rLetter) {
					num, err := strconv.Atoi(string(rLetter))
					checkErr(err)
					if !foundFirstNum {
						currXPoint = currXPoint*10 + num
					} else {
						currYPoint = currYPoint*10 + num
					}
				}
				if unicode.IsSpace(rLetter) {
					if foundFirstNum == false {
						foundFirstNum = true
					} else {
						break
					}
				}
				if !unicode.IsNumber(rLetter) && !unicode.IsSpace(rLetter) {
					break
				}
				index++
			}
		}

		if s == "Z" || s == "z" {
			shape.closedWithZ = true
		}

		index++
	}

	fmt.Println(shape.point)

	return shape
}

func InkUsed(shape *Shape) (ink int, err error) {
	ink = 0
	// get border length of shape - just add all the edges up!
	var edgeLength float64 = 0
	for i := 0; i < len(shape.edges); i++ {
		edgeLength += getLengthOfEdge(shape.edges[i])
	}
	// since int is an int, floor the edge lengths
	ink += int(math.Floor(edgeLength))
	if shape.filledIn {
		// if shape has non-transparent ink, need to find the area of it
		// meaning first we have to find if the shape produced by the edges is closed
		// todo: https://piazza.com/class/jbyh5bsk4ez3cn?cid=348 done with the assumption
		// the vote for "Simple, closed curve" will win.

	}
	return ink, nil
}

/*
1. First find if there's an intersection between the edges of the two polygons.
2. If not, then choose any one point of the first polygon and test whether it is fully inside the second.
3. If not, then choose any one point of the second polygon and test whether it is fully inside the first.
4. If not, then you can conclude that the two polygons are completely outside each other.
*/

func ShapesIntersect (A Shape, B Shape) bool {
	//1
	for i := 0; i < len(A.edges); i++ {
		for j := 0; j < len(B.edges); j++ {
			if EdgesIntersect(A.edges[i], B.edges[j]) {
				return true
			}
		}
	}
	//2
	pointA := A.edges[0].startPoint
	if pointInShape(pointA, B) {
		return true
	}
	//3
	pointB := B.edges[0].startPoint
	if pointInShape(pointB, A) {
		return true
	}
	//4
	return false
}

// https://martin-thoma.com/how-to-check-if-two-line-segments-intersect/
func EdgesIntersect(A Edge, B Edge) bool {

}

// https://en.wikipedia.org/wiki/Point_in_polygon
func pointInShape(point Point, shape Shape) bool {

}

func getLengthOfEdge(edge Edge) (length float64) {
	// a^2 + b^2 = c^2
	// a = horizontal length, b = vertical length
	a2b2 := math.Pow(float64((edge.startPoint.x - edge.endPoint.x)), 2) +
		math.Pow(float64((edge.startPoint.y - edge.endPoint.y)), 2)
	c := math.Sqrt(a2b2)
	return c
}