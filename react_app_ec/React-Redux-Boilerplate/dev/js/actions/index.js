export const renderCanvas = (canvas) => {
    return {
        type: 'RENDER_CANVAS',
        payload: canvas
    }
}

export const changeVersion = (version) => {
    return {
        type: 'RENDER_HISTORY',
        payload: version
    }
}