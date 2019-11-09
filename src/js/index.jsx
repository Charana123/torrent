import React from 'react'
import ReactDOM from 'react-dom'
import 'video.js/dist/video-js.min.css'
import 'bootstrap/dist/css/bootstrap.min.css'
import '../css/index.css'
import NavBar from './navbar.jsx'
import VideoPlayer from './video.jsx'
import Menu from './menu.jsx'

class Main extends React.Component {
    constructor(props){
        super(props)
        this.state = {
            playerData: null
        }
    }

    setPlayerData(torrent, fi){
        this.setState({
            playerData: {
                ctorrent: torrent,
                ctorrentFileIndex: fi,
            }
        })
    }
    
    render(){
        return (
            <React.Fragment>
                {/* <NavBar/> */}
                <VideoPlayer playerData={this.state.playerData}/>
                <Menu setPlayerData={this.setPlayerData.bind(this)}/>
            </React.Fragment>
        )
    }
}

ReactDOM.render(<Main/>, document.getElementById("root"));