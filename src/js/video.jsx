import React from 'react'
import videojs from 'video.js'
import '../css/video.css'

export default class VideoPlayer extends React.Component {
    
    constructor(props){
        super(props)
        this.state = {
            
        }
    }

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
            controls preload='auto' width='1800' data-setup='{}'>
                <p className='vjs-no-js'>
                    To view this video please enable JavaScript, and consider upgrading to a web browser that
                    <a href='https://videojs.com/html5-video-support/' target='_blank'>supports HTML5 video</a>
                </p>
            </video>
          </div>
          { this.props.playerData != null
                ? <div className="player-info">
                    <div className="current-torrent"> {this.props.playerData.ctorrent.Name} </div>
                    <div className="current-file">
                        <div className="video-icon fas fa-video"/>
                        <span className="mx-3"> 
                            {this.props.playerData.ctorrent.Files[this.props.playerData.ctorrentFileIndex].Name}
                        </span>
                    </div>
                </div>
                :  <div className="player-info"> </div>
          }
          <div className="underline"></div>
        </div>
      )
    }
}