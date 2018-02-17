package blockartlib

import (
	"testing"
	"fmt"
)



func setUpCanvas(xMax uint32, yMax uint32) {
	canvasT = &CanvasInstance{}
	canvasS := CanvasSettings{CanvasXMax:xMax, CanvasYMax:yMax}
	canvasT.settings = canvasS
}

func TestSvgToShape(t *testing.T) {
	// TEST happy path case
	setUpCanvas(100, 100)
	svgPath := "M 0 0 H 30 L 10 20 Z"
	shape, err := svgToShape(svgPath)
	edges := []Edge{}
	edges = append(edges, Edge{Start:Point{X:0, Y:0}, End:Point{X:30, Y:0}})
	edges = append(edges, Edge{Start:Point{X:30, Y:0}, End:Point{X:10, Y:20}})
	edges = append(edges, Edge{Start:Point{X:10, Y:20}, End:Point{X:0, Y:0}})
	if err != nil {
		t.Errorf("Error: %v \n", err)
	}
	if len(shape.Edges) != 3 {
		t.Errorf("Expected 3 edges, got %d \n", len(shape.Edges))
	}
	for _, edge := range edges {
		found := false
		for _, edgeFromPath := range shape.Edges {
			if edge == edgeFromPath {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Couldn't find edge %v in shape \n", edge)
		}
	}
	// TEST svg path too long error
	svgPath = " M 0 0 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 " +
		"H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 " +
			"H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 " +
				"H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 " +
					"H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 " +
						"H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 " +
							"H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 " +
								"H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 " +
									"H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 " +
										"H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 H 1 V 1 Z"
	shape, err = svgToShape(svgPath)
	if err != nil {
		switch err.(type) {
		case ShapeSvgStringTooLongError:
			break
		default:
			t.Errorf("Expected a ShapeSvgStringTooLongError error")
		}
	} else {
		t.Errorf("Expected a ShapeSvgStringTooLongError error \n")
	}

	svgPath = "M 0 0 H 30 M 0 10 L 30 40"
	shape, err = svgToShape(svgPath)
	edges = []Edge{}
	edges = append(edges, Edge{Start:Point{X:0, Y:0}, End:Point{X:30, Y:0}})
	edges = append(edges, Edge{Start:Point{X:0, Y:10}, End:Point{X:30, Y:40}})
	if err != nil {
		t.Errorf("Error: %v \n", err)
	}
	if len(shape.Edges) != 2 {
		t.Errorf("Expected 2 edges, got %d \n", len(shape.Edges))
	}
	for _, edge := range edges {
		found := false
		for _, edgeFromPath := range shape.Edges {
			if edge == edgeFromPath {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Couldn't find edge %v in shape \n", edge)
		}
	}
}

func TestSvgToCircleShape(t *testing.T) {
	shape, err := svgToCircleShape("1,2,5")
	if err != nil {
		fmt.Errorf("%v\n", err)
	}
	if !shape.IsCircle {
		fmt.Errorf("\n")
	}
	if !floatEquals(shape.Cx, 1) {
		fmt.Errorf("\n")
	}
	if !floatEquals(shape.Cy, 2) {
		fmt.Errorf("\n")
	}
	if !floatEquals(shape.Radius, 5) {
		fmt.Errorf("\n")
	}
}

func TestConvertShape(t *testing.T) {
	// TODO
}

func TestIsShapeInCanvas(t *testing.T) {
	setUpCanvas(100, 100)
	edges := []Edge{}
	edges = append(edges, Edge{Start:Point{X:0, Y:0}, End:Point{X:30, Y:0}})
	shape := Shape{Edges:edges}
	if !IsShapeInCanvas(shape) {
		t.Errorf("Edge {0,0}->{30,0} should be within canvas limits, is not \n")
	}
	shape.Edges = append(shape.Edges, Edge{Start:Point{0,100}, End:Point{100, 0}})
	if !IsShapeInCanvas(shape) {
		t.Errorf("Edge {0,100}->{100,0} should be within canvas limits, is not \n")
	}
	shape.Edges = append(shape.Edges, Edge{Start:Point{0, -1}, End:Point{1, 2}})
	if IsShapeInCanvas(shape) {
		t.Errorf("edge {0,-1}->{1,2} should not be within canvas limits, is \n")
	}
	//circle cases
	// fits in canvas
	circle := Shape{IsCircle:true, Cx:5, Cy:5, Radius:2}
	if !IsShapeInCanvas(circle) {
		t.Errorf("Circle with center (%d,%d) and radius %d should fit in canvas", circle.Cx, circle.Cy, circle.Radius)
	}
	// too large, circle encompasses canvas
	circle = Shape{IsCircle:true, Cx: 50, Cy: 50, Radius: 20000}
	if IsShapeInCanvas(circle) {
		t.Errorf("Circle with center (%d,%d) and radius %d should not fit in canvas", circle.Cx, circle.Cy, circle.Radius)
	}
	// circle encroaches one of the canvas' borders
	circle = Shape{IsCircle:true, Cx: 95, Cy: 20, Radius: 10}
	if IsShapeInCanvas(circle) {
		t.Errorf("Circle with center (%d,%d) and radius %d should not fit in canvas", circle.Cx, circle.Cy, circle.Radius)
	}
}

func TestInkUsed(t *testing.T) {
	// Case 1a: An open shape with a border
	var shape = Shape{}
	shape.FilledIn = false
	shape.BorderColor = "red"
	shape.Edges = []Edge{Edge{Start:Point{0,0}, End:Point{10,0}},
						Edge{Start:Point{10,0}, End:Point{10,10}},
						Edge{Start:Point{10,10}, End:Point{0,10}}}
	ink, err := InkUsed(&shape)
	if err != nil {
		t.Errorf("Received error %v \n", err)
	}
	if ink != 30 {
		t.Errorf("Expected 30 units of ink, used %d \n", ink)
	}
	// Case 1b: An open shape with no border and no fill
	shape.BorderColor = TRANSPARENT
	ink, err = InkUsed(&shape)
	if err == nil {
		t.Errorf("Expected error, received %d ink used\n", ink)
	}
	// Case 2: A closed shape, no fill, with a border
	shape.Edges = append(shape.Edges, Edge{Start:Point{0,10}, End:Point{0,0}})
	shape.BorderColor = "red"
	ink, err = InkUsed(&shape)
	if err != nil {
		t.Errorf("Received error %v \n", err)
	}
	if ink != 40 {
		t.Errorf("Expected 40 units of ink, used %d \n", ink)
	}
	// Case 3a: A closed shape, filled, with border
	shape.FilledIn = true
	ink, err = InkUsed(&shape)
	if err != nil {
		t.Errorf("Received error %v \n", err)
	}
	if ink != 140 {
		t.Errorf("Expected 140 units of ink, used %d \n", ink)
	}
	// Case 3b: A closed shape, filled, with no border
	shape.BorderColor = TRANSPARENT
	ink, err = InkUsed(&shape)
	if err != nil {
		t.Errorf("Received error %v \n", err)
	}
	if ink != 100 {
		t.Errorf("Expected 100 units of ink, used %d \n", ink)
	}
	// Case 4: Self-intersecting shape, no fill, with border
	shape.FilledIn = false
	shape.BorderColor = "red"
	shape.Edges = append(shape.Edges, Edge{Start:Point{0,0}, End:Point{10,10}})
	shape.Edges = append(shape.Edges, Edge{Start:Point{10,0}, End:Point{0,10}})
	ink, err = InkUsed(&shape)
	if err != nil {
		t.Errorf("Received error %v \n", err)
	}
	if ink != 68 {
		t.Errorf("Expected 68 units of ink, used %d \n", ink)
	}
	// Case 5: Self-intersecting shape, fill
	// Expect error
	shape.FilledIn = true
	ink, err = InkUsed(&shape)
	if err == nil {
		t.Errorf("Expected error when filledIn = true on self-intersecting shape \n")
	}

	// Circles - for these values just check the approximate value
	// Case 1: No border and no fill
	// Expect error
	circle := Shape{IsCircle:true, Cx: 5, Cy: 5, Radius: 8, FilledIn: false, BorderColor: TRANSPARENT}
	ink, err = InkUsed(&circle)
	if err == nil {
		t.Errorf("Expected error when both border and fill color is transparent")
	}
	// Case 2: Border, and no fill
	circle.BorderColor = "red"
	ink, err = InkUsed(&circle)
	if err != nil {
		t.Errorf("Received error %v \n", err)
	}
	if ink != 50 {
		t.Errorf("Expected 50 units of ink, received %d \n", ink)
	}
	// Case 3: No border, and fill
	circle.BorderColor = "transparent"
	circle.FilledIn = true
	circle.FillColor = "red"
	ink, err = InkUsed(&circle)
	if err != nil {
		t.Errorf("Received error %v \n", err)
	}
	if ink != 201 {
		t.Errorf("Expected 201 units of ink, received %d \n", ink)
	}
	// Case 4: Border, and fill
	circle.BorderColor = "red"
	ink, err = InkUsed(&circle)
	if err != nil {
		t.Errorf("Received error %v \n", err)
	}
	if ink != 251 {
		t.Errorf("Expected 201 units of ink, received %d \n", ink)
	}
}

func TestIsSimpleShape(t *testing.T) {
	// Case 1: Simple (non self intersecting)
	var shape = Shape{Edges:[]Edge{
		Edge{Start:Point{0,0}, End:Point{5,0}},
		Edge{Start:Point{5,0}, End:Point{5,5}},
		Edge{Start:Point{5,5}, End:Point{0,5}},
		Edge{Start:Point{0,0}}}}
	if !IsSimpleShape(&shape) {
		t.Errorf("Expected shape to be simple, isn't. \n")
	}
	// Case 2: Non-simple (self intersect)
	shape.Edges = append(shape.Edges, Edge{Start:Point{0,0}, End:Point{5,5}})
	shape.Edges = append(shape.Edges, Edge{Start:Point{0,5}, End:Point{5,0}})
	if IsSimpleShape(&shape) {
		t.Errorf("Expected shape to not be simple, is. \n")
	}
}

func TestGetAreaOfShape(t *testing.T) {
	// Case 1: Rectangle
	var rectangle = Shape{Edges:[]Edge{
		Edge{Start:Point{0,23}, End:Point{20,23}},
		Edge{Start:Point{20,23}, End:Point{20,8}},
		Edge{Start:Point{20,8}, End:Point{0,8}},
		Edge{Start:Point{0,8}, End:Point{0,23}}	}}
	area, err := getAreaOfShape(&rectangle)
	if err != nil {
		t.Errorf("Received error %v \n", err)
	}
	if !floatEquals(area, 300) {
		t.Errorf("Expected area of rectangle to be 300, received %f\n", area)
	}
	// Case 2: Triangle
	var triangle = Shape{Edges:[]Edge{
		Edge{Start:Point{0,8}, End:Point{20,8}},
		Edge{Start:Point{20,8}, End:Point{10,0}},
		Edge{Start:Point{10,0}, End:Point{0,8}}	}}
	area, err = getAreaOfShape(&triangle)
	if err != nil {
		t.Errorf("Received error %v \n", err)
	}
	if !floatEquals(area, 80) {
		t.Errorf("Expected area of triangle to be 80, received %f\n", area)
	}
	// Case 3: Pentagon
	var pentagon = Shape{Edges:[]Edge{
		Edge{Start:Point{0,23}, End:Point{20,23}},
		Edge{Start:Point{20,23}, End:Point{20,8}},
		Edge{Start:Point{20,8}, End:Point{10, 0}},
		Edge{Start:Point{10,0}, End:Point{0, 8}},
		Edge{Start:Point{0,8}, End:Point{0,23}}}}
	area, err = getAreaOfShape(&pentagon)
	if err != nil {
		t.Errorf("Received error %v \n", err)
	}
	if !floatEquals(area, 380) {
		t.Errorf("Expected area of pentagon to be 380, received %f\n", area)
	}
	// Case 4: Open shape, shouldn't be able to get area
	// Expect error
	var open = Shape{Edges:[]Edge {
		Edge{Start:Point{0,40}, End:Point{35,40}},
		Edge{Start:Point{35,40}, End:Point{35,0}}}}
	area, err = getAreaOfShape(&open)
	if err == nil {
		t.Errorf("Expected error, did not receive error. Area calculated: %f \n", area)
	}
}

func TestShapesIntersect(t *testing.T) {
	setUpCanvas(100, 100)
	// Case 1: Two line shapes intersect
	// Expect true
	var shape1 = Shape{}
	var shape2 = Shape{}
	shape1.Edges = []Edge{}
	shape1.Edges = append(shape1.Edges, Edge{Start:Point{0,10}, End:Point{10,10}})
	shape2.Edges = []Edge{}
	shape2.Edges = append(shape1.Edges, Edge{Start:Point{5,5}, End:Point{5,15}})
	if !ShapesIntersect(shape1, shape2, canvasT.settings) {
		t.Errorf("Shapes %v, %v should be intersecting \n", shape1, shape2)
	}
	// Case 2: A line and a rectangle intersect
	// Expect true
	shape1.Edges = append(shape1.Edges, Edge{Start:Point{10,10}, End:Point{10,20}})
	shape1.Edges = append(shape1.Edges, Edge{Start:Point{10,20}, End:Point{0,20}})
	shape1.Edges = append(shape1.Edges, Edge{Start:Point{0,20}, End:Point{0,10}})
	if !ShapesIntersect(shape1, shape2, canvasT.settings) {
		t.Errorf("Shapes %v, %v should be intersecting \n", shape1, shape2)
	}
	// Case 3: Two shapes don't intersect
	// Expect false
	shape2.Edges = []Edge{}
	shape2.Edges = append(shape2.Edges, Edge{Start:Point{0,40}, End:Point{35,40}})
	shape2.Edges = append(shape2.Edges, Edge{Start:Point{35,40}, End:Point{35,0}})
	if ShapesIntersect(shape1, shape2, canvasT.settings) {
		t.Errorf("Shapes %v, %v should not be intersecting \n", shape1, shape2)
	}
	// Case 4: A rectangle entirely within another rectangle
	// Expect true
	shape2.Edges = append(shape2.Edges, Edge{Start:Point{35,0}, End:Point{0,0}})
	shape2.Edges = append(shape2.Edges, Edge{Start:Point{0,0}, End:Point{0,40}})
	if !ShapesIntersect(shape1, shape2, canvasT.settings) {
		t.Errorf("Shapes %v, %v should be intersecting \n", shape1, shape2)
	}
	// Case 5: A rectangle entirely within another self-intersecting closed shape
	// Expect true
	shape2.Edges = append(shape2.Edges, Edge{Start:Point{0,40}, End:Point{10,25}})
	shape2.Edges = append(shape2.Edges, Edge{Start:Point{10,25}, End:Point{5,25}})
	shape2.Edges = append(shape2.Edges, Edge{Start:Point{5,25}, End:Point{5, 40}})
	if !ShapesIntersect(shape1, shape2, canvasT.settings) {
		t.Errorf("Shapes %v, %v should be intersecting \n", shape1, shape2)
	}

	// Circle-circle cases:
	// Two circles do not intersect
	circle1 := Shape{IsCircle:true, Cx:5, Cy:5, Radius:1}
	circle2 := Shape{IsCircle:true, Cx:10, Cy:10, Radius:1}
	if ShapesIntersect(circle1, circle2, canvasT.settings) {
		t.Errorf("Shapes %v, %v should not be intersecting \n", circle1, circle2)
	}
	// Two circles intersect
	circle1.Cx = 8
	circle1.Cy = 10
	circle1.Radius = 3
	circle2.Radius = 2
	if !ShapesIntersect(circle1, circle2, canvasT.settings) {
		t.Errorf("Shapes %v, %v should be intersecting \n", circle1, circle2)
	}
	// A circle encompasses another circle - transparent fill
	circle1.FilledIn = false
	circle2.FilledIn = false
	circle1.Cx = 10
	circle1.Cy = 10
	circle1.Radius = 10
	circle2.Cx = 5
	circle2.Cy = 5
	circle2.Radius = 1
	if ShapesIntersect(circle1, circle2, canvasT.settings) {
		t.Errorf("Shapes %v, %v should not be intersecting \n", circle1, circle2)
	}
	// A circle encompasses another circle - one colored, one transparent
	circle1.FilledIn = true
	circle2.FilledIn = false
	if ShapesIntersect(circle1, circle2, canvasT.settings) {
		t.Errorf("Shapes %v, %v should not be intersecting \n", circle1, circle2)
	}
	// A circle encompasses another circle - both colored fill
	circle1.FilledIn = true
	circle2.FilledIn = true
	if !ShapesIntersect(circle1, circle2, canvasT.settings) {
		t.Errorf("Shapes %v, %v should be intersecting \n", circle1, circle2)
	}
	// Circle-path cases:
	// Circle and path do not intersect
	fmt.Println("Test case 1")
	circle1.Cx = 5
	circle1.Cy = 5
	circle1.Radius = 3
	shape := Shape{IsCircle:false, Edges:[]Edge{
		Edge{Start:Point{X:50, Y:50}, End:Point{X:75, Y:50}}}}
	if ShapesIntersect(shape, circle1, canvasT.settings) {
		t.Errorf("Shapes %v, %v should not be intersecting \n", shape, circle1)
	}
	// Circle and path do intersect
	fmt.Println("Test case 2")
	shape.Edges = []Edge{Edge{Start:Point{2,5}, End:Point{5,5}}}
	if !ShapesIntersect(shape, circle1, canvasT.settings) {
		t.Errorf("Shapes %v, %v should be intersecting \n", shape, circle1)
	}
	// Path (open) is entirely within circle (transparent fill)
	fmt.Println("Test case 3")
	circle1.FilledIn = false
	circle1.Radius = 5
	shape.Edges = append(shape.Edges, Edge{Start:Point{5,5}, End:Point{6,6}})
	if ShapesIntersect(shape, circle1, canvasT.settings) {
		t.Errorf("Shapes %v, %v should not be intersecting \n", shape, circle1)
	}
	// Path(open) is entirely within circle (non-transparent fill)
	fmt.Println("Test case 4")
	circle1.FilledIn = true
	if !ShapesIntersect(shape, circle1, canvasT.settings) {
		t.Errorf("Shapes %v, %v should be intersecting \n", shape, circle1)
	}
	// Circle entirely within Path (this case a rectangle) (transparent fill)
	fmt.Println("Test case 5")
	shape.Edges = []Edge{Edge{Start:Point{0,0}, End:Point{50,0}},
		Edge{Start:Point{50,0}, End:Point{50,50}},
		Edge{Start:Point{50,50}, End:Point{0,50}},
		Edge{Start:Point{0,50}, End:Point{0,0}}}
	shape.FilledIn = false
	circle1.FilledIn = false
	if ShapesIntersect(shape, circle1, canvasT.settings) {
		t.Errorf("Shapes %v, %v should not be intersecting \n", shape, circle1)
	}
	// Circle entirely within rectangle (non-transparent fills)
	fmt.Println("Test case 6")
	shape.FilledIn = true
	circle1.FilledIn = true
	if !ShapesIntersect(shape, circle1, canvasT.settings) {
		t.Errorf("Shapes %v, %v should be intersecting \n", shape, circle1)
	}
	// Rectangle entirely within circle (transparent fill)
	fmt.Println("Test case 7")
	circle1.FilledIn = false
	shape.FilledIn = false
	circle1.Cx = 50
	circle1.Cy = 50
	circle1.Radius = 30
	shape.Edges = []Edge{Edge{Start:Point{40,40}, End:Point{40,42}},
		Edge{Start:Point{40,42}, End:Point{42,42}},
		Edge{Start:Point{42,42}, End:Point{42,40}},
		Edge{Start:Point{42,40}, End:Point{40,40}}}
	if ShapesIntersect(shape, circle1, canvasT.settings) {
		t.Errorf("Shapes %v, %v should not be intersecting \n", shape, circle1)
	}
	// Rectangle entirely within circle (non-transparent fills)
	fmt.Println("Test case 8")
	shape.FilledIn = true
	circle1.FilledIn = true
	if !ShapesIntersect(shape, circle1, canvasT.settings) {
		t.Errorf("Shapes %v, %v should be intersecting \n", shape, circle1)
	}
	// Rectangle partially in circle (transparent fill)
	fmt.Println("Test case 9")
	circle1.Cx = 10
	circle1.Cy = 10
	circle1.Radius = 3
	shape.Edges = []Edge{Edge{Start:Point{5,10}, End:Point{9,10}},
		Edge{Start:Point{9,10}, End:Point{9,15}},
		Edge{Start:Point{9,15}, End:Point{5,15}},
		Edge{Start:Point{5,15}, End:Point{5,10}}}
	circle1.FilledIn = false
	shape.FilledIn = false
	if ShapesIntersect(shape, circle1, canvasT.settings) {
		t.Errorf("Shapes %v, %v should not be intersecting \n", shape, circle1)
	}
	// Rectangle partially in circle (non-transparent fills)
	fmt.Println("Test case 10")
	circle1.FilledIn = true
	shape.FilledIn = true
	if !ShapesIntersect(shape, circle1, canvasT.settings) {
		t.Errorf("Shapes %v, %v should be intersecting \n", shape, circle1)
	}
}

func TestEdgesIntersect(t *testing.T) {
	// Case 1a: Two disjoint non-parallel lines, set flag to false
	// Expect false
	var edge1 = Edge{Start:Point{0,0}, End:Point{10,0}}
	var edge2 = Edge{Start:Point{0,10}, End:Point{30,40}}
	if EdgesIntersect(edge1, edge2, false) {
		t.Errorf("Edges %v, %v should not be intersecting \n", edge1, edge2)
	}
	// Case 1b: Two disjoint non-parallel lines, set flag to true
	// Expect false
	if EdgesIntersect(edge1, edge2, true) {
		t.Errorf("Edges %v, %v should not be intersecting \n", edge1, edge2)
	}
	// Case 2a: Two disjoint parallel lines, set flag to false
	// Expect false
	edge2 = Edge{Start:Point{0,10}, End:Point{30,10}}
	if EdgesIntersect(edge1, edge2, false) {
		t.Errorf("Edges %v, %v should not be intersecting \n", edge1, edge2)
	}
	// Case 2b: Two disjoint parallel lines, set flag to true
	// Expect false
	if EdgesIntersect(edge1, edge2, true) {
		t.Errorf("Edges %v, %v should not be intersecting \n", edge1, edge2)
	}
	// Case 3a: Two (clearly) intersecting lines, set flag to false
	// Expect true
	edge1 = Edge{Start:Point{0,10}, End:Point{30,40}}
	edge2 = Edge{Start:Point{0,20}, End:Point{30,20}}
	if !EdgesIntersect(edge1, edge2, false) {
		t.Errorf("Edges %v, %v should be intersecting \n", edge1, edge2)
	}
	// Case 3b: Two (clearly) intersecting lines, set flag to true
	// Expect true
	if !EdgesIntersect(edge1, edge2, true) {
		t.Errorf("Edges %v, %v should be intersecting \n", edge1, edge2)
	}
	// Case 4a: Two lines that touch only at the tips, set flag to true
	// Expect true
	edge2 = Edge{Start:Point{0, 10}, End:Point{0,5}}
	if !EdgesIntersect(edge1, edge2, true) {
		t.Errorf("Edges %v, %v should be intersecting \n", edge1, edge2)
	}
	// Case 4b: Two lines that touch only at the tips, set flag to false
	// Expect false
	if EdgesIntersect(edge1, edge2, false) {
		t.Errorf("Edges %v, %v should not be intersecting \n", edge1, edge2)
	}
	// Case 5a: Two parallel overlapping lines, set flag to false
	// Expect true
	edge1 = Edge{Start:Point{0,0}, End:Point{10,0}}
	edge2 = Edge{Start:Point{1,0}, End:Point{5,0}}
	if !EdgesIntersect(edge1, edge2, false) {
		t.Errorf("Edges %v, %v should be intersecting \n", edge1, edge2)
	}
	// Case 5b: Two parallel overlapping lines, set flag to true
	// Expect true
	if !EdgesIntersect(edge1, edge2, true) {
		t.Errorf("Edges %v, %v should be intersecting \n", edge1, edge2)
	}

	// Case 6:
	edge1 = Edge{Start:Point{0,0}, End:Point{5,0}}
	edge2 = Edge{Start:Point{5,0}, End:Point{5,5}}
	if EdgesIntersect(edge1, edge2, false) {
		t.Errorf("Edges %v, %v should not be intersecting \n", edge1, edge2)
	}
}

func TestOnlyIntersectsAtEndPoints(t *testing.T) {
	// Case 1: Only point1 touches edge
	// Expect true
	var edge = Edge{Start:Point{10,20}, End:Point{30,20}}
	var point1 = Point{10,20}
	var point2 = Point{0,20}
	var edgeB = Edge{Start:point1, End:point2}
	if !onlyIntersectsAtEndPoint(edge, edgeB) {
		t.Errorf("Expected edge %v to intersect edge with endpoints %v %v at one endpoint \n", edge, point1, point2)
	}
	// Case 2: Only point 2 touches edge
	// Expect true
	point1 = point2
	point2 = Point{10, 20}
	edgeB = Edge{Start:point1, End:point2}
	if !onlyIntersectsAtEndPoint(edge, edgeB) {
		t.Errorf("Expected edge %v to intersect edge with endpoints %v %v at one endpoint \n", edge, point1, point2)
	}
	// Case 3: Doesn't touch at all
	// Expect false
	point1 = Point{0, 20}
	point2 = Point{5,20}
	edgeB = Edge{Start:point1, End:point2}
	if onlyIntersectsAtEndPoint(edge, edgeB) {
		t.Errorf("Expected edge %v to not intersect edge with endpoints %v %v \n", edge, point1, point2)
	}

	// Case 4: Intersects at more than one place
	point1 = Point{0, 20}
	point2 = Point{20,20}
	edgeB = Edge{Start:point1, End:point2}
	if onlyIntersectsAtEndPoint(edge, edgeB) {
		t.Errorf("Expected edge %v to intersect edge with endpoints %v %v more than once \n", edge, point1, point2)
	}
}

func TestBoxesIntersect(t *testing.T) {
	// write these tests if bug encountered in one of its callers
}

func TestPointInShape(t *testing.T) {
	// write these tests if bug encountered in one of its callers

}

func TestPointsAreOnSameLine(t *testing.T) {
	// write these tests if bug encountered in one of its callers

}

func TestFindNextEdge(t *testing.T) {
	// write these tests if bug encountered in one of its callers
}