import React, {Component} from 'react';
import {connect} from 'react-redux';
import {renderCanvas} from '../actions/index'
import {bindActionCreators} from 'redux';
/*
 * We need "if(!this.props.user)" because we set state to null by default
 * */

class Canvas extends Component {

    constructor(props) {
        super(props);
    }

    render() {
        var width = this.props.canvas? this.props.canvas.width : 0; 
        var height = this.props.canvas? this.props.canvas.height : 0;
        var shapes = this.props.canvas? this.props.canvas.shapes : [];
        return (
            <div>
                <div style={{width: width, height: height}} className="canvas">
                </div>
                <button onClick={() => this.props.renderCanvas(shapes)}>Render</button>
                <code>{this.props.canvas? this.props.canvas.shapes : []}</code>
            </div>
        );
    }
}

function mapStateToProps(state) {
    return {
        canvas: state.canvas
    };
}

function matchDispatchToProps(dispatch){
    return bindActionCreators({renderCanvas: renderCanvas}, dispatch);
}

export default connect(mapStateToProps, matchDispatchToProps)(Canvas);
