
// "state = null" is set so that we don't throw an error when app first boots up
export default function (state = null, action) {
    switch (action.type) {
        case 'RENDER_CANVAS':
            var canvasShape = {}
            canvasShape.shapes = ["new shape", "cool"];
            canvasShape.width = 300
            canvasShape.height = 200
            return canvasShape;
            break;
    }
    return state;
}
