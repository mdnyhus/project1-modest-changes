package blockartlib

import (
	"crypto/ecdsa"
	"net/rpc"
	"math"
	"fmt"
	"crypto/md5"
	"encoding/hex"
	"strconv"
	"strings"
)

type CanvasInstance struct{
	canvasSettings CanvasSettings
	minerAddr string
	privKey ecdsa.PrivateKey
	client *rpc.Client
	settings CanvasSettings
}

const (
	MAX_SVG_LENGTH = 128
)

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

/*
	Trampoline function to call the parser, adds attributes after it parses
	@param: shapeType for extra credit
	@param: shape string for the svg
	@param: fill to determine if the svg shape needs to calculate area
	@param: width of the stroke
	@return: internal shape struct ; error otherwise
*/
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

/*
	Checking for errors and printing the context
	@param: string to describe the context of the error
	@param: err, error that is bubbled up
*/
func checkErr(context string, err error){
	if err != nil {
		fmt.Println(context)
		panic(err)
	}
}

/*
	Checking for errors and printing the context
	@param: svg path string
	@returns: boolean to if string is over
*/
func checkSvgStringLen(svgString string) bool {
	return len(svgString) > MAX_SVG_LENGTH
}


/*
	Trampoline to call parser, error detections
	@param: svg string for path
	@return: shape that is parsed with the internal struct or error otherwise
*/
func svgToShape(svgString string) (*Shape, error) {
	if checkSvgStringLen(svgString){
		return  nil, ShapeSvgStringTooLongError("Svg string has too many characters")
	}
	shape, err := ParseSvgPath2(svgString)
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
	@param: takes a shape assembled from the svg string, checks the list of edges' absolute points
	@return: boolean if all edges are within the canvas
*/
func svgIsInCanvas(shape Shape) bool {
	canvasXMax := float64(canvasT.settings.CanvasXMax)
	canvasYMax := float64(canvasT.settings.CanvasYMax)
	for _ , edge := range shape.edges{
		if !floatEquals(edge.startPoint.x, canvasXMax) && edge.startPoint.x > canvasXMax {
			return false
		}
		if !floatEquals(edge.startPoint.y, canvasYMax) && edge.startPoint.y > canvasYMax {
			return false
		}
		if !floatEquals(edge.endPoint.x, canvasXMax) && edge.endPoint.x > canvasXMax {
			return false
		}
		if !floatEquals(edge.endPoint.y, canvasYMax) && edge.endPoint.y > canvasYMax {
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
func hashShape(shape Shape) string {
	hasher := md5.New()
	s := fmt.Sprintf("%v", shape)
	hash := hasher.Sum([]byte(s))
	return hex.EncodeToString(hash)
}


/*
	Parses svg string to actual shape struct
		- splits the path by space and increment
	@param: d path of the svg string
	@return: shape that is filled with edges
*/
func ParseSvgPath2(path string)(*Shape, error) {
	args := strings.Split(path, " ")
	shape := Shape{}
	currentIndex := 0
	startPoint := Point{0.0, 0.0}
	currentPoint := Point{0.0, 0.0}
	for currentIndex < len(args) {
		arg := args[currentIndex]
		fmt.Println("The arguement " + strconv.Itoa(currentIndex) + " is: " + arg)
		isValid := checkOverFlow(currentIndex, arg, len(args))
		if !isValid{
			return nil , InvalidShapeSvgStringError("not valid string")
		}
		switch arg {
		case "M":
			handleMCase(&currentPoint, &startPoint, args[currentIndex + 1], args[currentIndex + 2], &currentIndex, true)
			break
		case "m":
			handleMCase(&currentPoint, &startPoint, args[currentIndex + 1], args[currentIndex + 2], &currentIndex, false)
			break
		case "L":
			handleLCase(&shape, &currentPoint, args[currentIndex + 1], args[currentIndex + 2], &currentIndex, true)
			break
		case "l":
			handleLCase(&shape, &currentPoint, args[currentIndex + 1], args[currentIndex + 2], &currentIndex, false)
			break
		case "V":
			handleVCase(&shape, &currentPoint, args[currentIndex + 1], &currentIndex, true)
			break
		case "v":
			handleVCase(&shape, &currentPoint, args[currentIndex + 1], &currentIndex, false)
			break
		case "H":
			handleHCase(&shape, &currentPoint, args[currentIndex + 1], &currentIndex, true)
			break
		case "h":
			handleHCase(&shape, &currentPoint, args[currentIndex + 1], &currentIndex, false)
			break
		case "z":
		case "Z":
			handleZCase(&shape, &currentPoint, &startPoint, &currentIndex)
			break
		default:
			return nil , InvalidShapeSvgStringError("not valid string")
		}
	}
	return &shape, nil
}

var TWONUMKEYWORDS = []string{"M", "m", "L", "l"}
var ONENUMKEYWORDS = []string{"V", "v", "H", "h"}


/*
	Checks if there is sufficient numbers after the keyword in the path
	@param: current index of the string
	@param: keyword: current keyword for paths
	@param: length of the entire svg string
	@return: the boolean if you have overflowed
*/
func checkOverFlow(index int , keyword string ,  length int) bool {
	offset := 0
	if containsKeyWord(TWONUMKEYWORDS, keyword){
		offset = 2
	} else if containsKeyWord(ONENUMKEYWORDS, keyword){
		offset = 1
	}
	return (index + offset) < length
}

/*
	Checks to see if it is a valid key word
	@param: array of keywords
	@param: keyword for svg path
	@return: the boolean if you have overflowed
*/
func containsKeyWord(s []string, e string) bool {
	for _, a := range s {
		if a == e {
			return true
		}
	}
	return false
}

/*
	Handles the M/m case, moves the current location of the pen, as well as creates a new start point
	@param: currentPoint: pointer to the current point (where the pen lies)
	@param: startPoint: the origin point (where the pen should go back to with z)
	@param: xVal: the x value for the svg
	@param: yVal: the y value for the svg
	@param: currentIndex: pointer to increment the val to next keyword
	@param: capital: to signal if capital keyword or not

*/
func handleMCase(currentPoint *Point,startPoint *Point, xVal string, yVal string, currentIndex *int, capital bool) {
	valX , err := strconv.ParseFloat(xVal, 64)
	checkErr("Not a valid float", err)
	valY , err := strconv.ParseFloat(yVal, 64)
	checkErr("Not a valid float", err)

	if capital{
		currentPoint.x = valX
		currentPoint.y = valY
	} else {
		currentPoint.x += valX
		currentPoint.y += valY
	}
	// new start origin for z close
	*startPoint = *currentPoint
	*currentIndex += 3
}

/*
	Handles the H/h case, adds a horizontal line
	@param: shape: the pointer to the current shape struct, adds to the list of edges
	@param: currentPoint: pointer to the current point (where the pen lies)
	@param: xVal: the x value for the svg
	@param: currentIndex: pointer to increment the val to next keyword
	@param: capital: to signal if capital keyword or not

*/
func handleHCase(shape *Shape, currentPoint *Point, xVal string, currentIndex *int, capital bool){
	valX , err := strconv.ParseFloat(xVal, 64)
	checkErr("Not a valid float", err)
	var endPoint Point
	if capital{
		endPoint = Point{valX, currentPoint.y}
	} else {
		endPoint = Point{currentPoint.x + valX, currentPoint.y}
	}
	edge := Edge{*currentPoint,endPoint }
	shape.edges = append(shape.edges, edge)
	*currentPoint = endPoint
	*currentIndex += 2
}

/*
	Handles the V/v case, adds a vertical line
	@param: shape: the pointer to the current shape struct, adds to the list of edges
	@param: currentPoint: pointer to the current point (where the pen lies)
	@param: yVal: the y value for the svg
	@param: currentIndex: pointer to increment the val to next keyword
	@param: capital: to signal if capital keyword or not

*/
func handleVCase(shape *Shape, currentPoint *Point, yVal string, currentIndex *int, capital bool) {
	valY , err := strconv.ParseFloat(yVal, 64)
	checkErr("Not a valid float", err)
	var endPoint Point

	if capital {
		endPoint = Point{currentPoint.x, valY}
	} else {
		endPoint = Point{currentPoint.x, currentPoint.y + valY}
	}
	edge := Edge{*currentPoint,endPoint }
	shape.edges = append(shape.edges, edge)
	*currentPoint = endPoint
	*currentIndex += 2
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
func handleLCase(shape *Shape, currentPoint *Point, xVal string, yVal string, currentIndex *int, capital bool) {
	valX , err := strconv.ParseFloat(xVal, 64)
	checkErr("Not a valid float", err)
	valY , err := strconv.ParseFloat(yVal, 64)
	checkErr("Not a valid float", err)

	var endPoint Point
	if capital{
		endPoint = Point{ valX, valY}
	} else {
		endPoint = Point{currentPoint.x + valX, currentPoint.y + valY}
	}

	edge := Edge{*currentPoint,endPoint }
	shape.edges = append(shape.edges, edge)
	*currentPoint = endPoint
	*currentIndex += 3
}

/*
	Handles the z/z case, closes off the shape from the origin point (not case sensitive)
	@param: shape: the pointer to the current shape struct, adds to the list of edges
	@param: currentPoint: pointer to the current point (where the pen lies)
	@param: startPoint: the origin point (where the pen should go back to with z)
	@param: currentIndex: pointer to increment the val to next keyword
*/

func handleZCase(shape *Shape, currentPoint *Point, startPoint *Point, currentIndex *int) {
	edge := Edge{*currentPoint,*startPoint}
	shape.edges = append(shape.edges, edge)
	*currentIndex += 1
}

// TODO
// - calculates the amount of ink required to draw the shape, in pixels
// @param shape *Shape: pointer to shape whose ink cost will be calculated
// @return ink int: amount of ink required to draw the shape
// @return error err: TODO
func InkUsed(shape *Shape) (ink int, err error) {
	var floatInk float64 = 0
	// get border length of shape - just add all the edges up!
	var edgeLength float64 = 0
	for i := 0; i < len(shape.edges); i++ {
		edgeLength += getLengthOfEdge(shape.edges[i])
	}
	// since ink is an int, floor the edge lengths
	floatInk += math.Floor(edgeLength)
	if shape.filledIn {
		// if shape has non-transparent ink, need to find the area of it
		// According to Ivan, if the shape has non-transparent ink, it'll be a simple closed shape
		// with no self-intersecting lines. So we can assume this will always be the case.
		floatInk += getAreaOfShape(shape)
	}
	ink = int(floatInk)
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
	MinX float64
	MinY float64
	MaxX float64
	MaxY float64
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
	var extendedX float64 = float64(settings.CanvasXMax)
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
func getCrossProduct(A Point, B Point) float64 {
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
func getAreaOfShape(shape *Shape) float64 {
	var start Edge = shape.edges[0]
	var area float64 = getCrossProduct(start.startPoint, start.endPoint)
	var current Edge = findNextEdge(shape, start)

	// keep looping until the "current" edge is the same as the start edge, you've found a cycle
	for ; current.startPoint.x != start.startPoint.x && current.startPoint.y != start.startPoint.y ; {
		area += getCrossProduct(current.startPoint, current.endPoint)
		current = findNextEdge(shape, current)
	}

	return math.Abs(area/2)
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