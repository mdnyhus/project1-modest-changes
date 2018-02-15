package blockartlib

import (
	"testing"
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
	edges = append(edges, Edge{start:Point{x:0, y:0}, end:Point{x:30, y:0}})
	edges = append(edges, Edge{start:Point{x:30, y:0}, end:Point{x:10, y:20}})
	edges = append(edges, Edge{start:Point{x:10, y:20}, end:Point{x:0, y:0}})
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
	edges = append(edges, Edge{start:Point{x:0, y:0}, end:Point{x:30, y:0}})
	edges = append(edges, Edge{start:Point{x:0, y:10}, end:Point{x:30, y:40}})
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

func TestConvertShape(t *testing.T) {
	// TODO
}

func TestSvgIsInCanvas(t *testing.T) {
	setUpCanvas(100, 100)
	edges := []Edge{}
	edges = append(edges, Edge{start:Point{x:0, y:0}, end:Point{x:30, y:0}})
	shape := Shape{Edges:edges}
	if !IsShapeInCanvas(shape) {
		t.Errorf("Edge {0,0}->{30,0} should be within canvas limits, is not \n")
	}
	shape.Edges = append(shape.Edges, Edge{start:Point{0,100}, end:Point{100, 0}})
	if !IsShapeInCanvas(shape) {
		t.Errorf("Edge {0,100}->{100,0} should be within canvas limits, is not \n")
	}
	shape.Edges = append(shape.Edges, Edge{start:Point{0, -1}, end:Point{1, 2}})
	if IsShapeInCanvas(shape) {
		t.Errorf("edge {0,-1}->{1,2} should not be within canvas limits, is \n")
	}
}

func TestInkUsed(t *testing.T) {
	// Case 1a: An open shape with a border
	var shape = Shape{}
	shape.FilledIn = false
	shape.BorderColor = "red"
	shape.Edges = []Edge{Edge{start:Point{0,0}, end:Point{10,0}},
						Edge{start:Point{10,0}, end:Point{10,10}},
						Edge{start:Point{10,10}, end:Point{0,10}}}
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
	shape.Edges = append(shape.Edges, Edge{start:Point{0,10}, end:Point{0,0}})
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
	shape.Edges = append(shape.Edges, Edge{start:Point{0,0}, end:Point{10,10}})
	shape.Edges = append(shape.Edges, Edge{start:Point{10,0}, end:Point{0,10}})
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
}

func TestIsSimpleShape(t *testing.T) {
	// Case 1: Simple (non self intersecting)
	var shape = Shape{Edges:[]Edge{
		Edge{start:Point{0,0}, end:Point{5,0}},
		Edge{start:Point{5,0}, end:Point{5,5}},
		Edge{start:Point{5,5}, end:Point{0,5}},
		Edge{start:Point{0,0}}}}
	if !isSimpleShape(&shape) {
		t.Errorf("Expected shape to be simple, isn't. \n")
	}
	// Case 2: Non-simple (self intersect)
	shape.Edges = append(shape.Edges, Edge{start:Point{0,0}, end:Point{5,5}})
	shape.Edges = append(shape.Edges, Edge{start:Point{0,5}, end:Point{5,0}})
	if isSimpleShape(&shape) {
		t.Errorf("Expected shape to not be simple, is. \n")
	}
}

func TestGetAreaOfShape(t *testing.T) {
	// Case 1: Rectangle
	var rectangle = Shape{Edges:[]Edge{
		Edge{start:Point{0,23}, end:Point{20,23}},
		Edge{start:Point{20,23}, end:Point{20,8}},
		Edge{start:Point{20,8}, end:Point{0,8}},
		Edge{start:Point{0,8}, end:Point{0,23}}	}}
	area, err := getAreaOfShape(&rectangle)
	if err != nil {
		t.Errorf("Received error %v \n", err)
	}
	if !floatEquals(area, 300) {
		t.Errorf("Expected area of rectangle to be 300, received %f\n", area)
	}
	// Case 2: Triangle
	var triangle = Shape{Edges:[]Edge{
		Edge{start:Point{0,8}, end:Point{20,8}},
		Edge{start:Point{20,8}, end:Point{10,0}},
		Edge{start:Point{10,0}, end:Point{0,8}}	}}
	area, err = getAreaOfShape(&triangle)
	if err != nil {
		t.Errorf("Received error %v \n", err)
	}
	if !floatEquals(area, 80) {
		t.Errorf("Expected area of triangle to be 80, received %f\n", area)
	}
	// Case 3: Pentagon
	var pentagon = Shape{Edges:[]Edge{
		Edge{start:Point{0,23}, end:Point{20,23}},
		Edge{start:Point{20,23}, end:Point{20,8}},
		Edge{start:Point{20,8}, end:Point{10, 0}},
		Edge{start:Point{10,0}, end:Point{0, 8}},
		Edge{start:Point{0,8}, end:Point{0,23}}}}
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
		Edge{start:Point{0,40}, end:Point{35,40}},
		Edge{start:Point{35,40}, end:Point{35,0}}}}
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
	shape1.Edges = append(shape1.Edges, Edge{start:Point{0,10}, end:Point{10,10}})
	shape2.Edges = []Edge{}
	shape2.Edges = append(shape1.Edges, Edge{start:Point{5,5}, end:Point{5,15}})
	if !ShapesIntersect(shape1, shape2, canvasT.settings) {
		t.Errorf("Shapes %v, %v should be intersecting \n", shape1, shape2)
	}
	// Case 2: A line and a rectangle intersect
	// Expect true
	shape1.Edges = append(shape1.Edges, Edge{start:Point{10,10}, end:Point{10,20}})
	shape1.Edges = append(shape1.Edges, Edge{start:Point{10,20}, end:Point{0,20}})
	shape1.Edges = append(shape1.Edges, Edge{start:Point{0,20}, end:Point{0,10}})
	if !ShapesIntersect(shape1, shape2, canvasT.settings) {
		t.Errorf("Shapes %v, %v should be intersecting \n", shape1, shape2)
	}
	// Case 3: Two shapes don't intersect
	// Expect false
	shape2.Edges = []Edge{}
	shape2.Edges = append(shape2.Edges, Edge{start:Point{0,40}, end:Point{35,40}})
	shape2.Edges = append(shape2.Edges, Edge{start:Point{35,40}, end:Point{35,0}})
	if ShapesIntersect(shape1, shape2, canvasT.settings) {
		t.Errorf("Shapes %v, %v should not be intersecting \n", shape1, shape2)
	}
	// Case 4: A rectangle entirely within another rectangle
	// Expect true
	shape2.Edges = append(shape2.Edges, Edge{start:Point{35,0}, end:Point{0,0}})
	shape2.Edges = append(shape2.Edges, Edge{start:Point{0,0}, end:Point{0,40}})
	if !ShapesIntersect(shape1, shape2, canvasT.settings) {
		t.Errorf("Shapes %v, %v should be intersecting \n", shape1, shape2)
	}
	// Case 5: A rectangle entirely within another self-intersecting closed shape
	// Expect true
	shape2.Edges = append(shape2.Edges, Edge{start:Point{0,40}, end:Point{10,25}})
	shape2.Edges = append(shape2.Edges, Edge{start:Point{10,25}, end:Point{5,25}})
	shape2.Edges = append(shape2.Edges, Edge{start:Point{5,25}, end:Point{5, 40}})
	if !ShapesIntersect(shape1, shape2, canvasT.settings) {
		t.Errorf("Shapes %v, %v should be intersecting \n", shape1, shape2)
	}
}

func TestEdgesIntersect(t *testing.T) {
	// Case 1a: Two disjoint non-parallel lines, set flag to false
	// Expect false
	var edge1 = Edge{start:Point{0,0}, end:Point{10,0}}
	var edge2 = Edge{start:Point{0,10}, end:Point{30,40}}
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
	edge2 = Edge{start:Point{0,10}, end:Point{30,10}}
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
	edge1 = Edge{start:Point{0,10}, end:Point{30,40}}
	edge2 = Edge{start:Point{0,20}, end:Point{30,20}}
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
	edge2 = Edge{start:Point{0, 10}, end:Point{0,5}}
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
	edge1 = Edge{start:Point{0,0}, end:Point{10,0}}
	edge2 = Edge{start:Point{1,0}, end:Point{5,0}}
	if !EdgesIntersect(edge1, edge2, false) {
		t.Errorf("Edges %v, %v should be intersecting \n", edge1, edge2)
	}
	// Case 5b: Two parallel overlapping lines, set flag to true
	// Expect true
	if !EdgesIntersect(edge1, edge2, true) {
		t.Errorf("Edges %v, %v should be intersecting \n", edge1, edge2)
	}

	// Case 6:
	edge1 = Edge{start:Point{0,0}, end:Point{5,0}}
	edge2 = Edge{start:Point{5,0}, end:Point{5,5}}
	if EdgesIntersect(edge1, edge2, false) {
		t.Errorf("Edges %v, %v should not be intersecting \n", edge1, edge2)
	}
}

func TestOnlyIntersectsAtEndPoints(t *testing.T) {
	// Case 1: Only point1 touches edge
	// Expect true
	var edge = Edge{start:Point{10,20}, end:Point{30,20}}
	var point1 = Point{10,20}
	var point2 = Point{0,20}
	var edgeB = Edge{start:point1, end:point2}
	if !onlyIntersectsAtEndPoint(edge, edgeB) {
		t.Errorf("Expected edge %v to intersect edge with endpoints %v %v at one endpoint \n", edge, point1, point2)
	}
	// Case 2: Only point 2 touches edge
	// Expect true
	point1 = point2
	point2 = Point{10, 20}
	edgeB = Edge{start:point1, end:point2}
	if !onlyIntersectsAtEndPoint(edge, edgeB) {
		t.Errorf("Expected edge %v to intersect edge with endpoints %v %v at one endpoint \n", edge, point1, point2)
	}
	// Case 3: Doesn't touch at all
	// Expect false
	point1 = Point{0, 20}
	point2 = Point{5,20}
	edgeB = Edge{start:point1, end:point2}
	if onlyIntersectsAtEndPoint(edge, edgeB) {
		t.Errorf("Expected edge %v to not intersect edge with endpoints %v %v \n", edge, point1, point2)
	}

	// Case 4: Intersects at more than one place
	point1 = Point{0, 20}
	point2 = Point{20,20}
	edgeB = Edge{start:point1, end:point2}
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