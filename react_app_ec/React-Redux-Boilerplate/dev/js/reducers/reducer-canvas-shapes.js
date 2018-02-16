
// "state = null" is set so that we don't throw an error when app first boots up

const initialState = {
    width : 0,
    height: 0, 
    shapeHistory: [],
    currentVersionCanvas: [],
}

export default function (previousState = initialState, action) {
    switch (action.type) {
        case 'RENDER_CANVAS':
            if (previousState.shapeHistory.length === 0 ){
                console.log("I am here----")
                console.log(action.payload)
                var newCanvasShape = {
                    width: 300,
                    height: 300,
                    shapeHistory: [action.payload],
                    currentVersionCanvas: action.payload}
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
                    shapeHistory: updatedHistory,
                    currentVersionCanvas: newestShapes
                }
                return newCanvasShape
            } else {
                console.log("nothing happened")
                return previousState
            }
            break;
        case 'RENDER_HISTORY':
            var currentVersion = action.payload
            var newCanvasShape = {
                width: previousState.width,
                height: previousState.height,
                shapeHistory: previousState.shapeHistory,
                currentVersionCanvas: []
            }
            return newCanvasShape
            break;
    }
    return previousState;
}
