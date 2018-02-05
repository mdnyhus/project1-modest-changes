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
	"crypto/md5"
	"encoding/hex"
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
	canvasInstance := CanvasInstance{}
	return canvasInstance, CanvasSettings{}, DisconnectedError("")
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
	- does an svg have one path, if not we can loop through all the matches
*/

func SvgToShape(svgString string) (Shape, error) {
	if checkSvgStringlen(svgString){
		return Shape{}, ShapeSvgStringTooLongError("Svg string has too many characters")
	}

	// getting the d-paths , include fill and other properties
	re, err  := regexp.Compile(" d=\".*\"\\/>")
	checkErr(err)
	matches := re.FindAllString(svgString , -1)
	if matches != nil {
		path := getDPoints(matches[0])
		shape, err := parseSvgPath(path)
		shape.svg = svgString
		shape.filledIn = checkIsFilled(matches[0])
		shape.hash = hashShape(shape)
		fmt.Println(shape)
		return shape , err
	}
	return Shape{}, InvalidShapeSvgStringError("not a valid shape")
}

/*
	Check if all the edges in the shape are within the campus
	// Todo
	// @param: takes a shape assembled from the svg string, and canvas settings
*/
func SvgIsInCanvas(shape Shape, settings CanvasSettings) bool {
	canvasXMax := int(settings.CanvasXMax)
	canvasYMax := int(settings.CanvasYMax)
	for _ , edge := range shape.edges{
 		if edge.startPoint.x > canvasXMax {
			return false
		}
		if edge.startPoint.y > canvasYMax {
			return false
		}
		if edge.endPoint.x > canvasXMax {
			return false
		}
		if edge.endPoint.y > canvasYMax {
			return false
		}
	}
	return true
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

func hashShape(shape Shape) string {
	hasher := md5.New()
	s := fmt.Sprintf("%v", shape)
	hash := hasher.Sum([]byte(s))
	return hex.EncodeToString(hash)
}

// only return the first d path and just the contents within the quotation marks
func getDPoints(svgPath string) string {
	re, err  := regexp.Compile("d=\".*?\"")
	checkErr(err)
	matches := re.FindAllString(svgPath , 1)
	if matches != nil {
		return matches[0]
	}
	return ""
}

// TODO: what should we error out, svg paths are someone error prone
//  - there can be many edge cases where an svg can be technically rendered
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

		// 2 Numbers following the keyword case
		if s == "M" || s == "m" || s == "L" || s == "l"{
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
					//not handing mid value letters
					break
				}
				index++
			}

			if s == "L" || s == "l" {
				startPoint := Point{currXPoint, currYPoint}
				var endPoint Point
				if s == "L" {
					endPoint.x = xPoint
					endPoint.y = yPoint
				} else if s == "l" {
					endPoint.x = currXPoint + xPoint
					endPoint.y = currYPoint + yPoint
				}
				currXPoint = endPoint.x
				currYPoint = endPoint.y
				edge := Edge{startPoint, endPoint}
				shape.edges = append(shape.edges, edge)
			} else {
				if s == "M" {
					currXPoint = xPoint
					currYPoint = yPoint
				} else if s == "m" {
					currXPoint = currXPoint + xPoint
					currYPoint = currYPoint + yPoint
				}
				originXPoint = currXPoint
				originYPoint = currYPoint
			}
		}

		// 1 Numbers following the keyword case
		if s == "H" || s == "V" || s == "h" || s == "v" {
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

		// special case Z
		if s == "Z" || s == "z" {
			edge := Edge{startPoint:Point{currXPoint, currYPoint}, endPoint:Point{originXPoint, originYPoint}}
			shape.edges = append(shape.edges, edge)
			shape.closedWithZ = true
		}
		// else move on
		index++
	}
	
	return shape , nil
}

// TODO
// - calculates the amount of ink required to draw the shape, in pixels
// @param shape *blockartlib.Shape: pointer to shape whose ink cost will be calculated
// @return ink int: amount of ink required to draw the shape
// @return error err: TODO
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
		// do this after the vote is completed and the criteria confirmed
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
	// 1: Do bounding boxes of each edge intersect?

	var boxA Box = buildBoundingBox(A)
	var boxB Box = buildBoundingBox(B)

	if !boxesIntersect(boxA, boxB) {
		return false
	}

	// 2: Does edge A intersect with edge segment B?
	// 2a: Check if the start or end point of B is on line A - this is for parallel lines
	var edgeA Edge = Edge{startPoint:Point{x:0, y:0},
		endPoint:Point{x:A.endPoint.x - A.startPoint.x, y:A.endPoint.y - A.startPoint.y}}
	var pointB1 Point = Point{x: B.startPoint.x - A.startPoint.x, y:B.startPoint.y - A.startPoint.y}
	var pointB2 Point = Point{x: B.endPoint.x - A.startPoint.x, y: B.endPoint.y - A.startPoint.y}
	if pointsAreOnOrigin(edgeA.endPoint, pointB1) || pointsAreOnOrigin(edgeA.endPoint, pointB2) {
		return true
	}
	// 2b: Check if the cross product of the start and end points of B with line A are of different signs
	// if they are, the lines intersect
	// https://stackoverflow.com/questions/7069420/check-if-two-line-segments-are-colliding-only-check-if-they-are-intersecting-n
	pointB1 = B.startPoint
	pointB2 = B.endPoint
	crossProduct1 := (A.endPoint.x - A.startPoint.x) * (pointB1.y - A.endPoint.y) -
		(A.endPoint.y - A.startPoint.y) * (pointB1.x - A.endPoint.x)
	crossProduct2 := (A.endPoint.x - A.startPoint.x) * (pointB2.y - A.endPoint.y) -
		(A.endPoint.y - A.startPoint.y) * (pointB2.x - A.endPoint.x)
	// if intersect, the signs of these cross products will be different
	return (crossProduct1 < 0 || crossProduct2 < 0) && !(crossProduct1 < 0 && crossProduct2 < 0)
}

type Box struct {
	MinX int
	MinY int
	MaxX int
	MaxY int
}

func buildBoundingBox(A Edge) Box {
	var boxA Box = Box{}
	if A.startPoint.x > A.endPoint.x {
		boxA.MaxX = A.startPoint.x
		boxA.MinX = A.endPoint.x
	} else {
		boxA.MaxX = A.endPoint.x
		boxA.MinX = A.startPoint.x
	}
	if A.startPoint.y > A.endPoint.y {
		boxA.MaxY = A.startPoint.y
		boxA.MinY = A.endPoint.y
	} else {
		boxA.MaxY = A.endPoint.y
		boxA.MinY = A.startPoint.y
	}
	return boxA
}

func boxesIntersect(A Box, B Box) bool {
	return A.MaxX >= B.MinX &&
		A.MinX <= B.MaxX &&
			A.MaxY >= B.MinY &&
				A.MinY <= B.MaxY
}

// https://www.geeksforgeeks.org/how-to-check-if-a-given-point-lies-inside-a-polygon/
func pointInShape(point Point, shape Shape) bool {
	//var extendX int = 100000 //todo: replace this number with what the canvas bound is, I can't find it at this moment
	//var edge Edge = Edge{startPoint:point, endPoint:Point{x:point.x + 1000000, y: point.y}}

	return false
}

func pointsAreOnOrigin(A Point, B Point) bool {
	return A.x * B.y - B.x * A.y == 0
}

func getLengthOfEdge(edge Edge) (length float64) {
	// a^2 + b^2 = c^2
	// a = horizontal length, b = vertical length
	a2b2 := math.Pow(float64((edge.startPoint.x - edge.endPoint.x)), 2) +
		math.Pow(float64((edge.startPoint.y - edge.endPoint.y)), 2)
	c := math.Sqrt(a2b2)
	return c
}