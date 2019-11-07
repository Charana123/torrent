import React from 'react'
import ReactDOM from 'react-dom'
import videojs from 'video.js'
import 'video.js/dist/video-js.css'
import 'bootstrap/dist/css/bootstrap.css'

function NavBar() {
    return (
        <nav className="navbar navbar-default sticky-top">
            <div className="container-fluid">
                <div className="navbar-header">
                    <a className="navbar-brand" href="#">WebSiteName</a>
                </div>
                <ul className="nav navbar-nav">
                <li className="active"><a href="#">Home</a></li>
                <li><a href="#">Page 1</a></li>
                <li><a href="#">Page 2</a></li>
                <li><a href="#">Page 3</a></li>
                </ul>
            </div>
        </nav>
    )
}

class VideoPlayer extends React.Component {
    
    // Instantiate Video.js on mount
    componentDidMount() {
      this.player = videojs(this.videoNode, this.props, function onPlayerReady() {
        console.log("Player Ready")
      });
    }
  
    // Destroy player on unmount
    componentWillUnmount() {
      if (this.player) {
        this.player.dispose()
      }
    }
  
    render() {
      return (
        <div>	
          <div data-vjs-player>
            <video ref={node => this.videoNode = node} className="video-js"
            controls preload='auto' width='1200' height='800' data-setup='{}'>
                <p className='vjs-no-js'>
                    To view this video please enable JavaScript, and consider upgrading to a web browser that
                    <a href='https://videojs.com/html5-video-support/' target='_blank'>supports HTML5 video</a>
                </p>
            </video>
          </div>
        </div>
      )
    }
}

function Index(){
    return (
        <React.Fragment>
            <NavBar/>
            <VideoPlayer/>
        </React.Fragment>
    )
}

ReactDOM.render(<Index/>, document.getElementById("root"));