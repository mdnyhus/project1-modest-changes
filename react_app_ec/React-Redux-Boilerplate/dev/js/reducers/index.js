import {combineReducers} from 'redux';
import CanvasReducer from './reducer-canvas-shapes'

/*
 * We combine all reducers into a single object before updated data is dispatched (sent) to store
 * Your entire applications state (store) is just whatever gets returned from all your reducers
 * */

const allReducers = combineReducers({
    canvas: CanvasReducer
});

export default allReducers
