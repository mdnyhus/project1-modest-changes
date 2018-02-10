package blockartlib

import (
	"crypto/ecdsa"
	"net/rpc"
	"math"
	"fmt"
	"crypto/md5"
	"encoding/hex"
	"strconv"
	"unicode"
	"strings"
	"unicode/utf8"
)

type CanvasInstance struct{
	canvasSettings CanvasSettings
	minerAddr string
	privKey ecdsa.PrivateKey
	client *rpc.Client
	settings CanvasSettings
}

// Public Methods
func (canvas CanvasInstance) AddShape(validateNum uint8, shapeType ShapeType, shapeSvgString string, fill string, stroke string) (shapeHash string, blockHash string, inkRemaining uint32, err error) {
	shape, e := convertShape(shapeType, shapeSvgString, fill, stroke)
	if e != nil {
		// TODO - deal with any errors convertShape may produce
		return shapeHash, blockHash, inkRemaining, e
	}

	args := &AddShapeArgs{
		Shape: *shape,
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

func (cavas CanvasInstance) GetSvgString(shapeHash string) (svgString string, err error){
	return "" , nil
}

func (canvas CanvasInstance) GetInk() (inkRemaining uint32, err error){
	return  0 , nil
}

func (canvas CanvasInstance) DeleteShape(validateNum uint8, shapeHash string) (inkRemaining uint32, err error) {
	return 0 , nil
}

func (canvas CanvasInstance) GetShapes(blockHash string) (shapeHashes []string, err error){
	return nil ,nil
}

func (cavas CanvasInstance) GetGenesisBlock() (blockHash string, err error){
	return "", nil
}

func (cavas CanvasInstance) GetChildren(blockHash string) (blockHashes []string, err error) {
	return nil, nil
}

func (canvas CanvasInstance) CloseCanvas() (inkRemaining uint32, err error){
	return 0, nil
}

// private methods
func convertShape(shapeType ShapeType, shapeSvgString string, fill string, stroke string) (*Shape, error){
	var shape *Shape
	var err error
	if shapeType == PATH {
		shape , err = svgToShape(shapeSvgString)
		if err != nil {
			return nil , err
		}
	}
	shape.svg = shapeSvgString
	shape.filledIn = strings.ToLower(fill) != "transparent"
	shape.filledInColor = fill
	shape.borderColor = stroke
	shape.hash = hashShape(*shape)
	return shape , nil
}



func checkErr(context string, err error){
	if err != nil {
		fmt.Println(context)
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

func svgToShape(svgString string) (*Shape, error) {
	if checkSvgStringlen(svgString){
		return  nil, ShapeSvgStringTooLongError("Svg string has too many characters")
	}
	shape, err := parseSvgPath(svgString)
	if err != nil {
		return nil, err
	}
	// check
	if !svgIsInCanvas(*shape){
		return nil , OutOfBoundsError(OutOfBoundsError{})
	}

	fmt.Println(shape)
	return shape , err
}

/*
	Check if all the edges in the shape are within the campus
	// Todo
	// @param: takes a shape assembled from the svg string, and canvas settings
*/
func svgIsInCanvas(shape Shape) bool {
	canvasXMax := int(canvasT.settings.CanvasXMax)
	canvasYMax := int(canvasT.settings.CanvasYMax)
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

func hashShape(shape Shape) string {
	hasher := md5.New()
	s := fmt.Sprintf("%v", shape)
	hash := hasher.Sum([]byte(s))
	return hex.EncodeToString(hash)
}

// TODO: what should we error out, svg paths are someone error prone
// TODO: deal with negative numbers
//  - there can be many edge cases where an svg can be technically rendered
func parseSvgPath(path string) (*Shape, error) {
	fmt.Println(path)
	shape := Shape{}
	currXPoint := 0
	currYPoint := 0
	index := 0

	originXPoint := 0
	originYPoint := 0

	period, _ := utf8.DecodeRuneInString(".")
	negative, _ := utf8.DecodeRuneInString("-")

	for {
		if index >= len(path) {
			break
		}

		char := path[index]
		s := string(char)
		fmt.Println("Position " + strconv.Itoa(index) + " looking at this char '" + s + "'")

		// 2 Numbers following the keyword case
		if s == "M" || s == "m" || s == "L" || s == "l"{
			xPoint := 0.0
			yPoint := 0.0
			onFirstNum := false
			onSecondNum := false
			finishedFirstNumber := false
			finishedSecondNumber := false
			negativeNum := false
			decimalMultiplier := 1.0

			for i := index + 1; i < len(path); i++ {
				letter := path[i]
				rLetter := rune(letter)
				if unicode.IsNumber(rLetter) {
					if !onFirstNum {
						onFirstNum = true
					} else if !onSecondNum {
						onSecondNum = true
					}
					if finishedSecondNumber {
						return nil, InvalidShapeSvgStringError("can not have more than three numbers behind M")
					}

					num, err := strconv.ParseFloat(string(rLetter), 64)
					checkErr("Couldn't convert string to float", err)
					if !finishedFirstNumber {
						if floatEquals(decimalMultiplier, 1.0) {
							if negativeNum {
								xPoint = xPoint * 10 - num
							} else {
								xPoint = xPoint*10 + num
							}
						} else {
							if negativeNum {
								xPoint -= num * decimalMultiplier
							} else {
								xPoint += num * decimalMultiplier
							}
							decimalMultiplier /= 10
						}
					} else {
						if floatEquals(decimalMultiplier, 1.0)  {
							if negativeNum {
								yPoint = yPoint*10 - num
							} else {
								yPoint = yPoint*10 + num
							}
						} else {
							if negativeNum {
								yPoint -= num * decimalMultiplier
							} else {
								yPoint += num * decimalMultiplier
							}
							decimalMultiplier /= 10
						}
					}
				}
				if unicode.IsSpace(rLetter) {
					if !finishedFirstNumber  && onFirstNum {
						// finished first number
						finishedFirstNumber = true
						decimalMultiplier = 1
						negativeNum = false
					} else if !finishedSecondNumber && onSecondNum {
						finishedSecondNumber = true
						decimalMultiplier = 1
						negativeNum = false
					}
				}
				if rLetter == period {
					if decimalMultiplier < 1 {
						return nil, InvalidShapeSvgStringError("Can't have more than one decimal point in number")
					}
					decimalMultiplier /= 10
				}
				if rLetter == negative {
					if s != "m" && s != "l" {
						return nil, InvalidShapeSvgStringError("Can't have negative numbers unless relative pathing")
					}
					if negativeNum {
						return nil, InvalidShapeSvgStringError("Can't have more than one negative sign in number")
					}
					negativeNum = true
				}
				if !unicode.IsNumber(rLetter) && !unicode.IsSpace(rLetter) &&
					rLetter != period && rLetter != negative {
					//not handing mid value letters
					break
				}
				index++
			}

			if s == "L" || s == "l" {
				startPoint := Point{currXPoint, currYPoint}
				var endPoint Point
				if s == "L" {
					endPoint.x = int(xPoint)
					endPoint.y = int(yPoint)
				} else if s == "l" {
					endPoint.x = currXPoint + int(xPoint)
					endPoint.y = currYPoint + int(yPoint)
				}
				currXPoint = endPoint.x
				currYPoint = endPoint.y
				edge := Edge{startPoint, endPoint}
				shape.edges = append(shape.edges, edge)
			} else {
				if s == "M" {
					currXPoint = int(xPoint)
					currYPoint = int(yPoint)
				} else if s == "m" {
					currXPoint = currXPoint + int(xPoint)
					currYPoint = currYPoint + int(yPoint)
				}
				originXPoint = currXPoint
				originYPoint = currYPoint
			}
		}

		// 1 Numbers following the keyword case
		if s == "H" || s == "V" || s == "h" || s == "v" {
			negativeNum := false
			decimalMultiplier := 1.0

			value := 0.0
			for i := index + 1; i < len(path); i++ {
				letter := path[i]
				rLetter := rune(letter)
				if unicode.IsNumber(rLetter) {
					num, err := strconv.ParseFloat(string(rLetter), 64)
					checkErr("Couldn't convert string to integer", err)
					if floatEquals(decimalMultiplier, 1.0) {
						if negativeNum {
							value = value * 10 - num
						} else {
							value = value * 10 + num
						}
					} else {
						if negativeNum {
							value -= num * decimalMultiplier
						} else {
							value += num * decimalMultiplier
						}
						decimalMultiplier /= 10
					}
				}
				if rLetter == negative {
					if negativeNum {
						return nil, InvalidShapeSvgStringError("Can't have two negatives")
					}
					if s != "h" && s != "v" {
						return nil, InvalidShapeSvgStringError("Can't use negative numbers in absolute paths")
					}
					negativeNum = true
				}
				if rLetter == period {
					if decimalMultiplier < 1 {
						return nil, InvalidShapeSvgStringError("Can't have two decimals in a number")
					}
					decimalMultiplier /= 10
				}
				if !unicode.IsNumber(rLetter) && !unicode.IsSpace(rLetter) &&
					rLetter != negative && rLetter != period {
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
					endPoint.x = int(value)
					currXPoint = int(value)
				} else {
					endPoint.x = currXPoint + int(value)
					currXPoint = currXPoint + int(value)
				}
				edge := Edge{startPoint, endPoint}
				shape.edges = append(shape.edges, edge)

			}
			if s == "V" || s == "v" {
				startPoint := Point{currXPoint, currYPoint}
				var endPoint Point
				endPoint.x = currXPoint
				if s == "V"{
					endPoint.y = int(value)
					currYPoint = int(value)
				} else {
					endPoint.y = currYPoint + int(value)
					currYPoint = currYPoint + int(value)
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

	return &shape, nil
}

// TODO
// - calculates the amount of ink required to draw the shape, in pixels
// @param shape *Shape: pointer to shape whose ink cost will be calculated
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
		// According to Ivan, if the shape has non-transparent ink, it'll be a simple closed shape
		// with no self-intersecting lines. So we can assume this will always be the case.
		ink += getAreaOfShape(shape)
	}
	return ink, nil
}

// @param A Shape
// @param B Shape
// @param canvasSettings CanvasSettings: Used to pass in the settings to the call to pointInShape
// @return bool
func ShapesIntersect (A Shape, B Shape, canvasSettings CanvasSettings) bool {
	/*
		1. First find if there's an intersection between the edges of the two polygons.
		2. If not, then choose any one point of the first polygon and test whether it is fully inside the second.
		3. If not, then choose any one point of the second polygon and test whether it is fully inside the first.
		4. If not, then you can conclude that the two polygons are completely outside each other.
	*/
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
	if pointInShape(pointA, B, canvasSettings) {
		return true
	}
	//3
	pointB := B.edges[0].startPoint
	if pointInShape(pointB, A, canvasSettings) {
		return true
	}
	//4
	return false
}

// Detects if two edges intersect
// @param A Edge
// @param B Edge
// @return bool
func EdgesIntersect(A Edge, B Edge) bool {
	// https://martin-thoma.com/how-to-check-if-two-line-segments-intersect/

	// 1: Do bounding boxes of each edge intersect?
	var boxA Box = buildBoundingBox(A)
	var boxB Box = buildBoundingBox(B)

	if !boxesIntersect(boxA, boxB) {
		return false
	}

	// 2: Does edge A intersect with edge segment B?
	// 2a: Check if the start or end point of B is on line A - this is for parallel lines
	// If cross product between two points is 0, it means the two points are on the same line through origin
	// meaning it is necessary to translate the edge to the origin, and the points of B accordingly
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

// Builds a bounding box for an edge. Private helper method for EdgesIntersect
// @param A Edge
// @return Box
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

// Checks if two boxes intersect. Private helper method for EdgesIntersect
// @param A Box
// @param B Box
// @return bool
func boxesIntersect(A Box, B Box) bool {
	// https://silentmatt.com/rectangle-intersection/
	return A.MaxX >= B.MinX &&
		A.MinX <= B.MaxX &&
		A.MaxY >= B.MinY &&
		A.MinY <= B.MaxY
}

// Checks if given point is in the given shape. Helper method for ShapesIntersect.
// Uses ray method - create a ray (horizontal edge) from point to edge of canvas,
// if the ray passes through an odd number of edges then the point is in the shape
// @param point Point
// @param shape Shape
// @param settings CanvasSettings: Get the max canvas limit for X
func pointInShape(point Point, shape Shape, settings CanvasSettings) bool {
	// https://www.geeksforgeeks.org/how-to-check-if-a-given-point-lies-inside-a-polygon/

	//var extendX int = 100000 //todo: replace this number with what the canvas bound is, I can't find it at this moment
	//var edge Edge = Edge{startPoint:point, endPoint:Point{x:point.x + 1000000, y: point.y}}
	var extendedX int = int(settings.CanvasXMax)
	var edge Edge = Edge{startPoint:point, endPoint:Point{x:extendedX, y:point.y}}
	// if this edge passes through an odd number of edges, the point is in shape
	intersects := 0
	for i := 0; i < len(shape.edges); i++ {
		if EdgesIntersect(edge, shape.edges[i]) {
			intersects++
		}
	}
	return intersects % 2 == 1
}

// Checks if the two points are on the origin. Private helper method for EdgesIntersect.
// @param A Point
// @param B Point
// @return bool
func pointsAreOnOrigin(A Point, B Point) bool {
	return getCrossProduct(A, B) == 0
}

// Gets cross product of two points
// @param A Point
// @param B Point
// @return int
func getCrossProduct(A Point, B Point) int {
	return A.x * B.y - B.x * A.y
}

// Gets length of an edge
// @param edge Edge
// @return float64
func getLengthOfEdge(edge Edge) float64 {
	// a^2 + b^2 = c^2
	// a = horizontal length, b = vertical length
	a2b2 := math.Pow(float64((edge.startPoint.x - edge.endPoint.x)), 2) +
		math.Pow(float64((edge.startPoint.y - edge.endPoint.y)), 2)
	c := math.Sqrt(a2b2)
	return c
}

// Gets area of shape by going through the edges until you've reached the beginning edge again.
// This function is assumed to be called only when calculating ink used for a non-transparent fill shape,
// which means the shape passed in is closed.
// @param shape *Shape
// @return int
func getAreaOfShape(shape *Shape) int {
	var start Edge = shape.edges[0]
	var area int = getCrossProduct(start.startPoint, start.endPoint)
	var current Edge = findNextEdge(shape, start)

	// keep looping until the "current" edge is the same as the start edge, you've found a cycle
	for ; current.startPoint.x != start.startPoint.x && current.startPoint.y != start.startPoint.y ; {
		area += getCrossProduct(current.startPoint, current.endPoint)
		current = findNextEdge(shape, current)
	}

	return int(math.Abs(float64(area)/2))
}

// Finds the next edge of the shape given current edge and the list of edges in shape
// @param shape *Shape
// @param edge Edge
// @return Edge
func findNextEdge(shape *Shape, edge Edge) Edge {
	var ret Edge
	for i := 0; i < len(shape.edges); i++ {
		if shape.edges[i].startPoint.x == edge.endPoint.x &&
			shape.edges[i].startPoint.y == edge.endPoint.y {
			ret = shape.edges[i]
			break
		}
	}
	return ret
}

// Checks for float equality by checking if the difference between the two floats is small
// @param a float64
// @param b float64
// @return bool
func floatEquals(a, b float64) bool {
	if ((a - b) < EPSILON && (b - a) < EPSILON) {
		return true
	}
	return false
}