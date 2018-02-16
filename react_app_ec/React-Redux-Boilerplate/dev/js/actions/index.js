export const renderCanvas = (canvas) => {
    return {
        type: 'RENDER_CANVAS',
        payload: canvas
    }
}