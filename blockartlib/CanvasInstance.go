package blockartlib

type CanvasInstance struct{
	canvasSettings CanvasSettings
}

func (canvas CanvasInstance) AddShape(validateNum uint8, shapeType ShapeType, shapeSvgString string, fill string, stroke string) (shapeHash string, blockHash string, inkRemaining uint32, err error) {

	return "" ,"" , 0, nil
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


