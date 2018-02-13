package blockartlib

import (
	"crypto/ecdsa"
	"crypto/md5"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"net/rpc"
	"sort"
	"strconv"
	"strings"
	"time"
)

type CanvasInstance struct {
	minerAddr      string
	privKey        ecdsa.PrivateKey
	client         *rpc.Client
	settings       CanvasSettings
}

const MAX_SVG_LENGTH = 128

var TwoNumKeyWords = []string{"M", "m", "L", "l"}
var OneNumKeyWords = []string{"V", "v", "H", "h"}

type Box struct {
	MinX float64
	MinY float64
	MaxX float64
	MaxY float64
}

// Public Methods
func (canvas CanvasInstance) AddShape(validateNum uint8, shapeType ShapeType, shapeSvgString string, fill string, stroke string) (shapeHash string, blockHash string, inkRemaining uint32, err error) {
	shape, err := convertShape(shapeType, shapeSvgString, fill, stroke)
	if err != nil {
		// TODO - deal with any errors convertShape may produce
		return shapeHash, blockHash, inkRemaining, err
	}

	hash := HashShape(*shape)
	shapeMeta := ShapeMeta{Hash: hash, Shape: *shape}

	args := &AddShapeArgs{
		ShapeMeta:   shapeMeta,
		ValidateNum: validateNum}
	var reply AddShapeReply
	if err = canvas.client.Call("LibMin.AddShape", args, &reply); err != nil {
		return hash, blockHash, inkRemaining, DisconnectedError(canvas.minerAddr)
	}

	return hash, reply.BlockHash, reply.InkRemaining, reply.Error
}

// Gets SVG string from the hashed shape
// @param canvas CanvasInstance
// @return string, error
func (canvas CanvasInstance) GetSvgString(shapeHash string) (svgString string, err error) {
	args := &GetSvgStringArgs{ShapeHash: shapeHash}
	var reply GetSvgStringReply
	if canvas.client.Call("LibMin.GetSvgString", args, &reply); err != nil {
		return svgString, DisconnectedError(canvas.minerAddr)
	}

	return reply.SvgString, reply.Error
}

// Gets ink remaining from canvas
// @param canvas CanvasInstance
// @return uint32, error
func (canvas CanvasInstance) GetInk() (inkRemaining uint32, err error) {
	// args are not used for GetInk
	var args int
	var reply uint32
	if canvas.client.Call("LibMin.GetInk", &args, &reply); err != nil {
		return inkRemaining, DisconnectedError(canvas.minerAddr)
	}

	return reply, nil
}

// Deletes shape from canvas and returns the new remaining ink count
// @param canvas CanvasInstance
// @return uint8, string
func (canvas CanvasInstance) DeleteShape(validateNum uint8, shapeHash string) (inkRemaining uint32, err error) {
	args := &DeleteShapeArgs{ValidateNum: validateNum, ShapeHash: shapeHash}
	var reply DeleteShapeReply
	if canvas.client.Call("LibMin.DeleteShape", args, &reply); err != nil {
		return inkRemaining, DisconnectedError(canvas.minerAddr)
	}

	return reply.InkRemaining, reply.Error
}

// Gets the shapes' hashes from a hashed block
// @param canvas CanvasInstance
// @return []string, error
func (canvas CanvasInstance) GetShapes(blockHash string) (shapeHashes []string, err error) {
	var reply GetShapesReply
	if canvas.client.Call("LibMin.GetShapes", &blockHash, &reply); err != nil {
		return shapeHashes, DisconnectedError(canvas.minerAddr)
	}

	return reply.ShapeHashes, reply.Error
}

// Gets the hash of the head block of the chain
// @param canvas CanvasInstance
// @return string, error
func (canvas CanvasInstance) GetGenesisBlock() (blockHash string, err error) {
	var reply string
	if canvas.client.Call("LibMin.GetGenesisBlock", nil, &reply); err != nil {
		return blockHash, DisconnectedError(canvas.minerAddr)
	}

	return reply, nil
}

// Gets children of a block in hashed format
// @param canvas CanvasInstance
// @return []string, error
func (canvas CanvasInstance) GetChildren(blockHash string) (blockHashes []string, err error) {
	var reply GetChildrenReply
	if canvas.client.Call("LibMin.GetChildren", &blockHash, &reply); err != nil {
		return blockHashes, DisconnectedError(canvas.minerAddr)
	}

	return reply.BlockHashes, reply.Error
}

// Close the canvas
// @param canvas CanvasInstance
// @return uint32, error
func (canvas CanvasInstance) CloseCanvas() (inkRemaining uint32, err error) {
	// TODO - stop any future operations on this canvas object
	// check https://piazza.com/class/jbyh5bsk4ez3cn?cid=428

	// get the ink remaining
	var reply uint32
	if canvas.client.Call("LibMin.GetInk", nil, &reply); err != nil {
		return inkRemaining, DisconnectedError(canvas.minerAddr)
	}

	return reply, nil
}

/*
	Wrapper function to call the parser, adds attributes after it parses
	@param: shapeType for extra credit
	@param: shape string for the svg
	@param: fill to determine if the svg shape needs to calculate area
	@param: width of the stroke
	@return: internal shape struct ; error otherwise
*/
func convertShape(shapeType ShapeType, shapeSvgString string, fill string, stroke string) (*Shape, error) {
	var shape *Shape
	if shapeType == PATH {
		var err error
		shape, err = svgToShape(shapeSvgString)
		shape, err = svgToShape(shapeSvgString)
		if err != nil {
			return nil, err
		}
	}
	shape.Svg = shapeSvgString
	shape.FilledIn = strings.ToLower(fill) != TRANSPARENT
	shape.FillColor = fill
	shape.BorderColor = stroke
	shape.Timestamp = time.Now()
	return shape, nil
}

/*
	Checking for errors and printing the context
	@param: svg path string
	@returns: true if string is too long, false otherwise
*/
func IsSvgTooLong(svgString string) bool {
	return len(svgString) > MAX_SVG_LENGTH
}

/*
	Wrapper to call parser, error detections
	@param: svg string for path
	@return: shape that is parsed with the internal struct or error otherwise
*/
func svgToShape(svgString string) (*Shape, error) {
	if IsSvgTooLong(svgString) {
		return nil, ShapeSvgStringTooLongError(svgString)
	}
	shape, err := ParseSvgPath(svgString)
	if err != nil {
		return nil, err
	}
	if !IsShapeInCanvas(*shape) {
		return nil, InvalidShapeSvgStringError(svgString)
	}
	return shape, err
}

/*
	Check if all the edges in the shape are within the campus
	@param: takes a shape assembled from the svg string, checks the list of edges' absolute points
	@return: boolean if all edges are within the canvas
*/
func IsShapeInCanvas(shape Shape) bool {
	canvasXMax := float64(canvasT.settings.CanvasXMax)
	canvasYMax := float64(canvasT.settings.CanvasYMax)
	for _, edge := range shape.Edges {
		if edge.start.x < 0 || edge.start.y < 0 || edge.end.x < 0 || edge.end.y < 0 {
			return false
		}

		if !floatEquals(edge.start.x, canvasXMax) && edge.start.x > canvasXMax {
			return false
		}
		if !floatEquals(edge.start.y, canvasYMax) && edge.start.y > canvasYMax {
			return false
		}
		if !floatEquals(edge.end.x, canvasXMax) && edge.end.x > canvasXMax {
			return false
		}
		if !floatEquals(edge.end.y, canvasYMax) && edge.end.y > canvasYMax {
			return false
		}
	}
	return true
}

/*
	Uses md5 and hashes the shape
	@param: shape
	@return: hash of the shape
*/
func HashShape(shape Shape) string {
	sorted := shape
	sort.Sort(Edges(sorted.Edges))
	hasher := md5.New()
	s := fmt.Sprintf("%v", sorted)
	hash := hasher.Sum([]byte(s))
	return hex.EncodeToString(hash)
}

/*
	Parses svg string to actual shape struct
		splits the path by space and increment
		currPoint = the current position where the pen is when drawing the svg
		start point = the point where the pen is moved to, for z cases
		@param: path string: path of the svg string
	@return: shape that is filled with edges
*/
func ParseSvgPath(path string) (*Shape, error) {
	args := strings.Split(path, " ")
	shape := Shape{}
	currIndex := 0
	startPoint := Point{0.0, 0.0}
	currPoint := Point{0.0, 0.0}
	var parseError error = nil
	for currIndex < len(args) {
		arg := args[currIndex]
		argUpper := strings.ToUpper(arg)
		isValid := hasSufficientArgs(currIndex, arg, len(args))
		if !isValid {
			return nil, InvalidShapeSvgStringError("Not valid string: " + path)
		}
		switch argUpper {
		case "M":
			err := handleMCase(&currPoint, &startPoint, args[currIndex+1], args[currIndex+2], arg == argUpper)
			parseError = err
			currIndex += 3
		case "L":
			err := handleLCase(&shape, &currPoint, args[currIndex+1], args[currIndex+2], arg == argUpper)
			parseError = err
			currIndex += 3
		case "V":
			err := handleVCase(&shape, &currPoint, args[currIndex+1], arg == argUpper)
			parseError = err
			currIndex += 2
		case "H":
			err := handleHCase(&shape, &currPoint, args[currIndex+1], arg == argUpper)
			parseError = err
			currIndex += 2
		case "Z":
			handleZCase(&shape, &currPoint, &startPoint)
			currIndex += 1
		default:
			return nil, InvalidShapeSvgStringError("not valid string")
		}
		if parseError != nil {
			return nil, InvalidShapeSvgStringError("not valid string")
		}
	}
	return &shape, nil
}

/*
	Checks if there is sufficient numbers after the keyword in the path
	@param: current index of the string
	@param: keyword: current keyword for paths
	@param: length of the entire svg string
	@return: the boolean if you have overflowed
*/
func hasSufficientArgs(index int, keyword string, length int) bool {
	offset := getOffsetFromKeyword(keyword)
	return (index + offset) < length
}

/*
	Checks to see if it is a valid key word
	@param: array of keywords
	@param: keyword for svg path
	@return: the boolean if you have overflowed
*/
func getOffsetFromKeyword(keyWord string) int {
	for _, word := range OneNumKeyWords {
		if word == keyWord {
			return 1
		}
	}
	for _, word := range TwoNumKeyWords {
		if word == keyWord {
			return 2
		}
	}
	return 0
}

/*
	Handles the M/m case, moves the current location of the pen, as well as creates a new start point
	@param: currentPoint: pointer to the current point (where the pen lies)
	@param: start: the origin point (where the pen should go back to with z)
	@param: xVal: the x value for the svg
	@param: yVal: the y value for the svg
	@param: currentIndex: pointer to increment the val to next keyword
	@param: capital: to signal if capital keyword or not

*/
func handleMCase(currentPoint *Point, startPoint *Point, xVal string, yVal string, capital bool) error {
	valX, err := strconv.ParseFloat(xVal, 64)
	if err != nil {
		return err
	}
	valY, err := strconv.ParseFloat(yVal, 64)
	if err != nil {
		return err
	}

	if capital {
		currentPoint.x = valX
		currentPoint.y = valY
	} else {
		currentPoint.x += valX
		currentPoint.y += valY
	}
	// new start origin for z close
	*startPoint = *currentPoint
	return nil
}

/*
	Handles the L/l case, adds a line to the edge
	@param: shape: the pointer to the current shape struct, adds to the list of edges
	@param: currentPoint: pointer to the current point (where the pen lies)
	@param: xVal: the x value for the svg
	@param: yVal: the y value for the svg
	@param: currentIndex: pointer to increment the val to next keyword
	@param: capital: to signal if capital keyword or not

*/
func handleLCase(shape *Shape, currentPoint *Point, xVal string, yVal string, capital bool) error {
	valX, err := strconv.ParseFloat(xVal, 64)
	if err != nil {
		return err
	}
	valY, err := strconv.ParseFloat(yVal, 64)
	if err != nil {
		return err
	}

	var endPoint Point
	if capital {
		endPoint = Point{valX, valY}
	} else {
		endPoint = Point{currentPoint.x + valX, currentPoint.y + valY}
	}

	edge := Edge{*currentPoint, endPoint}
	shape.Edges = append(shape.Edges, edge)
	*currentPoint = endPoint
	return nil
}

/*
	Handles the V/v case, adds a vertical line
	@param: shape: the pointer to the current shape struct, adds to the list of edges
	@param: currentPoint: pointer to the current point (where the pen lies)
	@param: yVal: the y value for the svg
	@param: currentIndex: pointer to increment the val to next keyword
	@param: capital: to signal if capital keyword or not

*/
func handleVCase(shape *Shape, currentPoint *Point, yVal string, capital bool) error {
	valY, err := strconv.ParseFloat(yVal, 64)
	if err != nil {
		return err
	}
	var endPoint Point

	if capital {
		endPoint = Point{currentPoint.x, valY}
	} else {
		endPoint = Point{currentPoint.x, currentPoint.y + valY}
	}
	edge := Edge{*currentPoint, endPoint}
	shape.Edges = append(shape.Edges, edge)
	*currentPoint = endPoint
	return nil
}

/*
	Handles the H/h case, adds a horizontal line
	@param: shape: the pointer to the current shape struct, adds to the list of edges
	@param: currentPoint: pointer to the current point (where the pen lies)
	@param: xVal: the x value for the svg
	@param: currentIndex: pointer to increment the val to next keyword
	@param: capital: to signal if capital keyword or not

*/
func handleHCase(shape *Shape, currentPoint *Point, xVal string, capital bool) error {
	valX, err := strconv.ParseFloat(xVal, 64)
	if err != nil {
		return err
	}
	var endPoint Point
	if capital {
		endPoint = Point{valX, currentPoint.y}
	} else {
		endPoint = Point{currentPoint.x + valX, currentPoint.y}
	}
	edge := Edge{*currentPoint, endPoint}
	shape.Edges = append(shape.Edges, edge)
	*currentPoint = endPoint
	return nil
}

/*
	Handles the Z/z case, closes off the shape from the origin point (not case sensitive)
	@param: shape: the pointer to the current shape struct, adds to the list of edges
	@param: currentPoint: pointer to the current point (where the pen lies)
	@param: start: the origin point (where the pen should go back to with z)
	@param: currentIndex: pointer to increment the val to next keyword
*/

func handleZCase(shape *Shape, currentPoint *Point, startPoint *Point) {
	edge := Edge{*currentPoint, *startPoint}
	shape.Edges = append(shape.Edges, edge)
}

// - calculates the amount of ink required to draw the shape, in pixels
// @param shape *Shape: pointer to shape whose ink cost will be calculated
// @return ink int: amount of ink required to draw the shape
// @return error err
func InkUsed(shape *Shape) (ink uint32, err error) {
	var floatInk float64 = 0
	if shape.FilledIn {
		// if shape has non-transparent ink, need to find the area of it
		// According to Ivan, if the shape has non-transparent ink, it'll be a simple closed shape
		// with no self-intersecting lines. So we can assume this will always be the case.
		// We need to check if the shape passed in is in fact simple and closed
		area, err := getAreaOfShape(shape)
		if err != nil {
			floatInk += area
		} else {
			return 0, err
		}
		if !isSimpleShape(shape) {
			return 0, errors.New("Can't have non-transparent ink if shape has self-intersecting edges")
		}
	} else {
		// get border length of shape - just add all the edges up!
		var edgeLength float64 = 0
		for _, edge := range shape.Edges {
			edgeLength += getLengthOfEdge(edge)
		}
		floatInk = edgeLength
	}
	ink = uint32(floatInk)
	return ink, nil
}

// Checks if shape is closed and simple
// @param shape Shape
// @return bool
func isSimpleShape(shape *Shape) bool {
	// Check if the edges don't self-intersect ("simple")
	for i := 0; i < len(shape.Edges); i++ {
		for j := i + 1; j < len(shape.Edges); j++ {
			if EdgesIntersect(shape.Edges[i], shape.Edges[j], false) {
				return false
			}
		}
	}
	return true
}

// Gets area of shape by going through the edges until you've reached the beginning edge again.
// This function is assumed to be called only when calculating ink used for a non-transparent fill shape,
// which means the shape passed in is closed.
// @param shape *Shape
// @return int
func getAreaOfShape(shape *Shape) (float64, error) {
	// https://www.mathopenref.com/coordpolygonarea.html
	var start Edge = shape.Edges[0]
	var area float64 = getCrossProduct(start.start, start.end)
	current, err := findNextEdge(shape, start)
	if err != nil {
		return 0, errors.New("Couldn't find area of an open shape")
	}

	// keep looping until the "current" edge is the same as the start edge, you've found a cycle
	for current.start.x != start.start.x && current.start.y != start.start.y {
		area += getCrossProduct(current.start, current.end)
		current, err = findNextEdge(shape, *current)
		if err != nil {
			return 0, errors.New("Couldn't find area of an open shape")
		}
	}

	return math.Abs(area / 2), nil
}

// @param A Shape
// @param B Shape
// @param canvasSettings CanvasSettings: Used to pass in the settings to the call to pointInShape
// @return bool
func ShapesIntersect(A Shape, B Shape, canvasSettings CanvasSettings) bool {
	//1. First find if there's an intersection between the edges of the two polygons.
	for _, edgeA := range A.Edges {
		for _, edgeB := range B.Edges {
			if EdgesIntersect(edgeA, edgeB, true) {
				return true
			}
		}
	}
	//2. If not, then choose any one point of the first polygon and test whether it is fully inside the second.
	pointA := A.Edges[0].start
	if pointInShape(pointA, B, canvasSettings) {
		return true
	}
	//3. If not, then choose any one point of the second polygon and test whether it is fully inside the first.
	pointB := B.Edges[0].start
	if pointInShape(pointB, A, canvasSettings) {
		return true
	}
	//4. If not, then you can conclude that the two polygons are completely outside each other.
	return false
}

// Detects if two edges intersect
// @param A Edge
// @param B Edge
// @return bool
func EdgesIntersect(A Edge, B Edge, countTipToTipIntersect bool) bool {
	// https://martin-thoma.com/how-to-check-if-two-line-segments-intersect/

	// 1: Check if each edge intersect
	var boxA Box = buildBoundingBox(A)
	var boxB Box = buildBoundingBox(B)

	if !boxesIntersect(boxA, boxB) {
		return false
	}

	// 2: Check if edge A intersects with edge segment B
	// 2a: Check if the start or end point of B is on line A - this is for parallel lines
	// If cross product between two points is 0, it means the two points are on the same line through origin
	// meaning it is necessary to translate the edge to the origin, and the points of B accordingly
	var edgeA Edge = Edge{start: Point{x: 0, y: 0},
		end: Point{x: A.end.x - A.start.x, y: A.end.y - A.start.y}}
	var pointB1 Point = Point{x: B.start.x - A.start.x, y: B.start.y - A.start.y}
	var pointB2 Point = Point{x: B.end.x - A.start.x, y: B.end.y - A.start.y}
	if pointsAreOnSameLine(edgeA.end, pointB1) || pointsAreOnSameLine(edgeA.end, pointB2) {
		if !countTipToTipIntersect {
		// if the endpoints are the only ones touching the edge, don't return true
			if !onlyIntersectsAtEndPoint(A, pointB1, pointB2) {
				return true
			}
		} else {
			return true
		}
	}
	// 2b: Check if the cross product of the start and end points of B with line A are of different signs
	// if they are, the lines intersect
	// https://stackoverflow.com/questions/7069420/check-if-two-line-segments-are-colliding-only-check-if-they-are-intersecting-n
	pointB1 = B.start
	pointB2 = B.end
	//A.x * B.y - B.x * A.y
	crossProduct1 := getCrossProduct(Point{x: A.end.x - A.start.x, y: pointB1.x - A.end.x},
		Point{x: A.end.y - A.start.y, y: pointB1.y - A.end.y})
	crossProduct2 := getCrossProduct(Point{x: A.end.x - A.start.x, y: pointB2.x - A.end.x},
		Point{x: A.end.y - A.start.y, y: pointB2.y - A.end.y})
	// if intersect, the signs of these cross products will be different
	return (crossProduct1 < 0 || crossProduct2 < 0) && !(crossProduct1 < 0 && crossProduct2 < 0)
}

// Checks if the two lines (B represented by its endpoints)
// only intersect at one of its tips. Private helper function for EdgesIntersect.
// ASSUME only used for parallel lines (only used in this context in caller)
// @param A Edge
// @param pointB1 Point
// @param pointB2 Point
// @return bool
func onlyIntersectsAtEndPoint(edge Edge, pointB1 Point, pointB2 Point) bool {
	// to account for corner cases of horizontal/vertical lines,
	// have to check if line is more "vertical" or "horizontal"
	xDelt := math.Abs(edge.start.x - edge.end.x)
	yDelt := math.Abs(edge.start.y - edge.end.y)
	if pointB1 == edge.start || pointB1 == edge.end {
		// have to check if pointB2 is somewhere along the A line
		if xDelt > yDelt {
			return !(pointB2.x >= edge.start.x && pointB2.x <= edge.end.x ||
				pointB2.x >= edge.end.x && pointB2.x <= edge.start.x)
		} else {
			return !(pointB2.y >= edge.start.y && pointB2.y <= edge.end.y ||
				pointB2.y >= edge.end.y && pointB2.y <= edge.start.y)
		}
	} else if pointB2 == edge.start || pointB2 == edge.end {
		// have to check if pointB1 is somewhere along A line
		if xDelt > yDelt {
			return !(pointB1.x >= edge.start.x && pointB1.x <= edge.end.x ||
				pointB1.x >= edge.end.x && pointB1.x <= edge.start.x)
		} else {
			return !(pointB1.y >= edge.start.y && pointB1.y <= edge.end.y ||
				pointB1.y >= edge.end.y && pointB1.y <= edge.start.y)
		}
	}
	return false
}

// Builds a bounding box for an edge. Private helper method for EdgesIntersect
// @param A Edge
// @return Box
func buildBoundingBox(edge Edge) Box {
	var boxA Box = Box{}
	if edge.start.x > edge.end.x {
		boxA.MaxX = edge.start.x
		boxA.MinX = edge.end.x
	} else {
		boxA.MaxX = edge.end.x
		boxA.MinX = edge.start.x
	}
	if edge.start.y > edge.end.y {
		boxA.MaxY = edge.start.y
		boxA.MinY = edge.end.y
	} else {
		boxA.MaxY = edge.end.y
		boxA.MinY = edge.start.y
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

	var extendedX = float64(settings.CanvasXMax)
	var edge = Edge{start: point, end: Point{x: extendedX, y: point.y}}
	// if this edge passes through an odd number of edges, the point is in shape
	intersections := 0
	for _, e := range shape.Edges {
		if EdgesIntersect(edge, e, true) {
			intersections++
		}
	}
	return intersections%2 == 1
}

// Checks if the two points are on the origin. Private helper method for EdgesIntersect.
// @param A Point
// @param B Point
// @return bool
func pointsAreOnSameLine(A Point, B Point) bool {
	return getCrossProduct(A, B) < EPSILON
}

// Gets cross product of two points
// @param A Point
// @param B Point
// @return int
func getCrossProduct(A Point, B Point) float64 {
	return A.x*B.y - B.x*A.y
}

// Gets length of an edge
// @param edge Edge
// @return float64
func getLengthOfEdge(edge Edge) float64 {
	// a^2 + b^2 = c^2
	// a = horizontal length, b = vertical length
	a2b2 := math.Pow(float64(edge.start.x-edge.end.x), 2) +
		math.Pow(float64(edge.start.y-edge.end.y), 2)
	c := math.Sqrt(a2b2)
	return c
}

// Finds the next edge of the shape given current edge and the list of edges in shape
// @param shape *Shape
// @param edge Edge
// @return Edge
func findNextEdge(shape *Shape, edge Edge) (*Edge, error) {
	var ret *Edge
	for _, e := range shape.Edges {
		if e.start.x == edge.end.x &&
			e.start.y == edge.end.y {
			ret = &e
			return ret, nil
		}
	}
	return nil, errors.New("Couldn't find next edge") // todo: find a better error?
}

// Checks for float equality by checking if the difference between the two floats is small
// @param a float64
// @param b float64
// @return bool
func floatEquals(a, b float64) bool {
	return (a-b) < EPSILON && (b-a) < EPSILON
}
