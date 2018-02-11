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
}

func TestSvgIsInCanvas(t *testing.T) {

}

func TestInkUsed(t *testing.T) {

}

func TestIsSimpleShape(t *testing.T) {

}

func TestGetAreaOfShape(t *testing.T) {

}

func TestShapesIntersect(t *testing.T) {

}

func TestEdgesIntersect(t *testing.T) {

}

func TestOnlyIntersectsAtEndPoints(t *testing.T) {

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