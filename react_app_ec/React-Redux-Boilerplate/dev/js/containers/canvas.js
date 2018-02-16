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
        var width = this.props.canvas.width;
        var height = this.props.canvas.height;
        var htmlShapes
        if (this.props.canvas.shapeHistory.length !== 0){
            var shapeHistory = this.props.canvas.shapeHistory;
            var currentVersion = shapeHistory.length - 1;
            var shapes = shapeHistory[currentVersion]
            htmlShapes = shapes.map((shape, index) =>
                <div key={index} dangerouslySetInnerHTML={{__html: shape}}/>
            );
        }
        return (
            <div className="container">
                <div className="row">
                    <div className="col-md">
                        <div id="canvas" style={{width: width, height: height}} className="canvas">
                            {htmlShapes}
                        </div>
                        <button 
                        className="btn btn-primary"
                        onClick={() => this.props.renderCanvas(['<svg height="210" width="400"><path d="M150 0 L75 200 L225 200 Z" /></svg>','<svg height="210" width="400"><path d="M250 0 L75 200 L225 200 Z" /></svg>'])}>Render</button>
                    </div>
                    <div className="col-md">
                        <h1>History</h1>
                    </div>
                </div>
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
