
// "state = null" is set so that we don't throw an error when app first boots up

const initialState = {
    width : 0,
    height: 0, 
    shapeHistory: []
}


export default function (previousState = initialState, action) {
    switch (action.type) {
        case 'RENDER_CANVAS':
            if (previousState.shapeHistory.length === 0 ){
                var newCanvasShape = {
                    width: 300,
                    height: 300,
                    shapeHistory: [action.payload]}
                return newCanvasShape
            }
            var newestShapes = action.payload;
            var currentVersion = previousState.shapeHistory.length - 1;
            var shapesInCurrentVersion = previousState.shapeHistory[currentVersion];
            var numShapesInCurrentVersion = shapesInCurrentVersion.length;
                
            // something happened
            if (newestShapes.length != numShapesInCurrentVersion) {
                // must copy since reducers have to be pure 
                var updatedHistory = previousState.shapeHistory.concat([newestShapes]);
                var newCanvasShape = {
                    width: previousState.width,
                    height: previousState.height,
                    shapeHistory: updatedHistory}
                return newCanvasShape
            } else {
                console.log("nothing happened")
                return previousState
            }
            break;
    }
    return previousState;
}
