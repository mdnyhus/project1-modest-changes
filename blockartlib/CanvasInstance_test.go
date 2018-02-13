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
	if len(shape.edges) != 3 {
		t.Errorf("Expected 3 edges, got %d \n", len(shape.edges))
	}
	for _, edge := range edges {
		found := false
		for _, edgeFromPath := range shape.edges {
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
	if len(shape.edges) != 2 {
		t.Errorf("Expected 2 edges, got %d \n", len(shape.edges))
	}
	for _, edge := range edges {
		found := false
		for _, edgeFromPath := range shape.edges {
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
	shape := Shape{edges:edges}
	if !svgIsInCanvas(shape) {
		t.Errorf("Edge {0,0}->{30,0} should be within canvas limits, is not \n")
	}
	shape.edges = append(shape.edges, Edge{start:Point{0,100}, end:Point{100, 0}})
	if !svgIsInCanvas(shape) {
		t.Errorf("Edge {0,100}->{100,0} should be within canvas limits, is not \n")
	}
	shape.edges = append(shape.edges, Edge{start:Point{0, -1}, end:Point{1, 2}})
	if svgIsInCanvas(shape) {
		t.Errorf("edge {0,-1}->{1,2} should not be within canvas limits, is \n")
	}
}

func TestInkUsed(t *testing.T) {

}

func TestIsSimpleShape(t *testing.T) {

}

func TestGetAreaOfShape(t *testing.T) {

}

func TestShapesIntersect(t *testing.T) {
	setUpCanvas(100, 100)
	// Case 1: Two line shapes intersect
	// Expect true
	var shape1 = Shape{}
	var shape2 = Shape{}
	shape1.edges = []Edge{}
	shape1.edges = append(shape1.edges, Edge{start:Point{0,10}, end:Point{10,10}})
	shape2.edges = []Edge{}
	shape2.edges = append(shape1.edges, Edge{start:Point{5,5}, end:Point{5,15}})
	if !ShapesIntersect(shape1, shape2, canvasT.settings) {
		t.Errorf("Shapes %v, %v should be intersecting \n", shape1, shape2)
	}
	// Case 2: A line and a rectangle intersect
	// Expect true
	shape1.edges = append(shape1.edges, Edge{start:Point{10,10}, end:Point{10,20}})
	shape1.edges = append(shape1.edges, Edge{start:Point{10,20}, end:Point{0,20}})
	shape1.edges = append(shape1.edges, Edge{start:Point{0,20}, end:Point{0,10}})
	if !ShapesIntersect(shape1, shape2, canvasT.settings) {
		t.Errorf("Shapes %v, %v should be intersecting \n", shape1, shape2)
	}
	// Case 3: Two shapes don't intersect
	// Expect false
	shape2.edges = []Edge{}
	shape2.edges = append(shape2.edges, Edge{start:Point{0,40}, end:Point{35,40}})
	shape2.edges = append(shape2.edges, Edge{start:Point{35,40}, end:Point{35,0}})
	if ShapesIntersect(shape1, shape2, canvasT.settings) {
		t.Errorf("Shapes %v, %v should not be intersecting \n", shape1, shape2)
	}
	// Case 4: A rectangle entirely within another rectangle
	// Expect true
	shape2.edges = append(shape2.edges, Edge{start:Point{35,0}, end:Point{0,0}})
	shape2.edges = append(shape2.edges, Edge{start:Point{0,0}, end:Point{0,40}})
	if !ShapesIntersect(shape1, shape2, canvasT.settings) {
		t.Errorf("Shapes %v, %v should be intersecting \n", shape1, shape2)
	}
	// Case 5: A rectangle entirely within another self-intersecting closed shape
	// Expect true
	shape2.edges = append(shape2.edges, Edge{start:Point{0,40}, end:Point{10,25}})
	shape2.edges = append(shape2.edges, Edge{start:Point{10,25}, end:Point{5,25}})
	shape2.edges = append(shape2.edges, Edge{start:Point{5,25}, end:Point{5, 40}})
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
}

func TestOnlyIntersectsAtEndPoints(t *testing.T) {
	// Case 1: Only point1 touches edge
	// Expect true
	var edge = Edge{start:Point{10,20}, end:Point{30,20}}
	var point1 = Point{10,20}
	var point2 = Point{0,20}
	if !onlyIntersectsAtEndPoint(edge, point1, point2) {
		t.Errorf("Expected edge %v to intersect edge with endpoints %v %v at one endpoint \n", edge, point1, point2)
	}
	// Case 2: Only point 2 touches edge
	// Expect true
	point1 = point2
	point2 = Point{10, 20}
	if !onlyIntersectsAtEndPoint(edge, point1, point2) {
		t.Errorf("Expected edge %v to intersect edge with endpoints %v %v at one endpoint \n", edge, point1, point2)
	}
	// Case 3: Doesn't touch at all
	// Expect false
	point1 = Point{0, 20}
	point2 = Point{5,20}
	if onlyIntersectsAtEndPoint(edge, point1, point2) {
		t.Errorf("Expected edge %v to not intersect edge with endpoints %v %v \n", edge, point1, point2)
	}

	// Case 4: Intersects at more than one place
	point1 = Point{0, 20}
	point2 = Point{20,20}
	if onlyIntersectsAtEndPoint(edge, point1, point2) {
		t.Errorf("Expected edge %v to intersect edge with endpoints %v %v more than once \n", edge, point1, point2)
	}
}

func TestBoxesIntersect(t *testing.T) {

}

func TestPointInShape(t *testing.T) {

}

func TestPointsAreOnSameLine(t *testing.T) {

}

func TestFindNextEdge(t *testing.T) {
	//closedShape := Shape{}
}