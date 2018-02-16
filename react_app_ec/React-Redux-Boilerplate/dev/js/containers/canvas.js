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
        var historyList
        if (this.props.canvas.shapeHistory.length !== 0){
            var shapeHistory = this.props.canvas.shapeHistory;
            var currentVersion = shapeHistory.length - 1;
            var shapes = shapeHistory[currentVersion]
            htmlShapes = appendSvgPaths(shapeHistory[currentVersion])
            historyList = shapeHistory.map((version, index)=> 

                <li key={index}>
                    <a onClick={()=>{console.log("sup")}}>Version: {index + 1}</a>
                </li>    
            );
        }
        return (
            <div className="container">
                <div className="row">
                    <div className="col-md-6">
                        <h1>Canvas</h1>
                        <div id="canvas" style={{width: width, height: height}} className="canvas">
                            <div dangerouslySetInnerHTML={{__html: htmlShapes}} />
                        </div>
                        <button 
                        className="btn btn-primary"
                        onClick={() => this.props.renderCanvas(['<path d="M50 0 L25 20 L225 200 Z" />','<path d="M250 0 L75 200 L225 200 Z" />'])}>Render</button>
                    </div>
                    <div className="col-md-6">
                        <h1>History</h1>
                        {historyList}
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

function appendSvgPaths(listOfSvgPaths){
    var svg = "";
    for (var i = 0; i < listOfSvgPaths.length; i++ ){
        svg += listOfSvgPaths[i]
    }
    return "<svg>" + svg + "</svg>"
}

// need api stub

function matchDispatchToProps(dispatch){
    return bindActionCreators({renderCanvas: renderCanvas}, dispatch);
}

export default connect(mapStateToProps, matchDispatchToProps)(Canvas);
