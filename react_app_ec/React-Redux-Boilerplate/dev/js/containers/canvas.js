import React, {Component} from 'react';
import {connect} from 'react-redux';
import {renderCanvas, changeVersion} from '../actions/index'
import {bindActionCreators} from 'redux';
/*
 * We need "if(!this.props.user)" because we set state to null by default
 * */

class Canvas extends Component {

    constructor(props) {
        super(props);
    }

    render() {
        var canvas = this.props.canvas
        var width = canvas.width;
        var height = canvas.height;
        var htmlShapes
        var historyList
        if (this.props.canvas.shapeHistory.length !== 0){
            var shapeHistory = canvas.shapeHistory;
            var currentVersion = shapeHistory.length - 1;
            htmlShapes = appendSvgPaths(canvas.currentVersionCanvas)
            historyList = shapeHistory.map((version, index)=> 
                <li key={index}>
                    <a onClick={()=>{this.props.changeVersion({})}}>Version: {index + 1}</a>
                </li>    
            );
        }
        return (
            <div className="container">
                <div className="row">
                    <div className="col-md-6">
                        <h1>Canvas</h1>
                        <button 
                        className="btn btn-primary"
                        onClick={callMinerServer.bind(this)}>Update</button>
                        <div id="canvas" style={{width: width, height: height}} className="canvas">
                            <div dangerouslySetInnerHTML={{__html:htmlShapes}} />
                        </div>
                       
                    </div>
                    {/* <div className="col-md-6">
                        <h1>History</h1>
                        {historyList}
                    </div> */}
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

function appendSvgPaths(listOfSvgPaths) {
    var svg = "";
    for (var i = 0; i < listOfSvgPaths.length; i++ ){
        svg += listOfSvgPaths[i]
    }
    return "<svg>" + svg + "</svg>"
}

// need api stub

function callMinerServer(){
    console.log("calling miner server")
    console.log(this)
    var url = "http://localhost:8080/"
    $.ajax({
        url: url,
        data: "nothing",
        success: updateCanvas.bind(this),
        dataType: "json",
        crossDomain: true,
      }).fail(function (err){
        console.log("error")
        console.log(err)
      });

}

function updateCanvas(data, status, jqXHR){
    //dummy 
    var d = data
    var xMax = d.CanvasXMax
    var yMax = d.CanvasYMax
    var shapes = d.SvgStrings

    this.props.renderCanvas(d)
}

function matchDispatchToProps(dispatch){
    return bindActionCreators({renderCanvas: renderCanvas, changeVersion:changeVersion }, dispatch);
}

export default connect(mapStateToProps, matchDispatchToProps)(Canvas);
