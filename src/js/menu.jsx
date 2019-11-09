import React from 'react'
import Toggle from 'react-toggle'
import 'react-toggle/style.css'
import '../css/menu.css'
import { CircularProgressbarWithChildren } from 'react-circular-progressbar'
import 'react-circular-progressbar/dist/styles.css';

class PlayVideo extends React.Component {
    render(){
        return (
            <div className="float-left">
                <div className="stacked-container">
                    <img className="play-video-img"/>
                    <div className="play-video-progressbar">
                        <CircularProgressbarWithChildren value={66}>
                            <i className="fas fa-play"></i>
                        </CircularProgressbarWithChildren>
                    </div>
                </div>
            </div>
        )
    }
}

class TorrentInfo extends React.Component {
    constructor(props){
        super(props)
        this.state = {
            ShowFiles: false,
        }
    }

    render(){
        return (
            <React.Fragment>
            <div className="mt-4"> 
                <span> { this.props.torrent.Name } </span>
                { this.state.ShowFiles
                    ? <span className="mr-5 float-right fas fa-chevron-up"> </span>
                    : <span className="mr-5 float-right fas fa-chevron-down"> </span>
                }
                <span className="mr-5 float-right fas fa-trash"></span>
            </div>
            <span className="mt-3 torrent-info-status-2">
                <Toggle
                    defaultChecked={this.props.torrent.State.On}
                    onChange={(event) => this.props.handleStateOn(event, ti)} />
                { this.props.torrent.State.Seeding 
                    ? <span className="ml-3">Seeding</span>
                    : <span className="ml-3">Downloading</span>}
                <span className="ml-3"> | </span>
                { this.props.torrent.State.Seeding
                    ? <span className="ml-3 fas fa-upload"></span>
                    : <span className="ml-3 fas fa-download"></span>}
                <span className="ml-3"> { this.props.torrent.Speed } </span>
            </span>
            </React.Fragment>
        )
    }
 
}

function FileMenu(props) {
    return (
        <React.Fragment>
        { props.torrent.Files.map((file, fi)=> {
            return (
                <div key={file.Name}
                    onClick={() => props.setPlayerData(torrent, fi)}> 
                    {file.Name} 
                </div>
            )
        }) }
        </React.Fragment>
    )
}

export default class Menu extends React.Component {
    constructor(props){
        super(props)
        this.state = {
            torrents: [
                {
                    Name: "[ConnectPal] Allison Parker Pack [19 clips][2017]",
                    Speed: "0 B/s",
                    DateAdded: "11/7/2019",
                    NumFiles: "19 Files",
                    TotalSize: "1 GB",
                    Files: [
                        { Name: "Allison Parker 1.mp4" },
                        { Name: "Allison Parker 2.mp4" },
                        { Name: "Allison Parker 3.mp4" },
                        { Name: "Allison Parker 4.mp4" },
                        { Name: "Allison Parker 5.mp4" },
                        { Name: "Allison Parker 6.mp4" },
                        { Name: "Allison Parker 7.mp4" },
                        { Name: "Allison Parker 8.mp4" },
                        { Name: "Allison Parker 9.mp4" },
                        { Name: "Allison Parker 10.mp4" },
                        { Name: "Allison Parker 11.mp4" },
                        { Name: "Allison Parker 12.mp4" },
                        { Name: "Allison Parker 13.mp4" },
                        { Name: "Allison Parker 14.mp4" },
                        { Name: "Allison Parker 15.mp4" },
                        { Name: "Allison Parker 16.mp4" },
                        { Name: "Allison Parker 17.mp4" },
                        { Name: "Allison Parker 18.mp4" },
                        { Name: "Allison Parker 19.mp4" },
                    ],
                    State: {
                        Seeding: false,
                        On: false,
                    }
                },
            ]
        }
    }

    handleStateOn(event, ti){
        console.log("handle state on")
        // send download/seeding state on/off request
        // get torrent information
    }

    render(){
        return (
            <div className="ml-5">
                <div className="mt-3">
                    <button className="add-torrent-btn btn btn-primary" type="button">
                        <span class="mr-2 fas fa-plus"></span>
                        Add Torrent
                    </button>
                </div>
                <div>
                { 
                    this.state.torrents.map((torrent, ti) => {
                        return (
                            <div className="menu-item" key={torrent.Name}>
                                <div className="row">
                                    <div className="menu-item-left-column col-auto ml-5"> 
                                        <PlayVideo/>
                                    </div>
                                    <div className="col ml-5">
                                        <TorrentInfo
                                            handleStateOn={this.handleStateOn.bind(this)}
                                            torrent={torrent}/>
                                    </div>
                                </div>
                                <div className="row">
                                    <div className="mt-3 menu-item-left-column col-auto ml-5">
                                        <div className="mt-3"> Date Added </div>
                                        <div> {torrent.DateAdded} </div>
                                        <div className="mt-3"> Total Files </div>
                                        <div> {torrent.NumFiles} </div>
                                        <div className="mt-3"> Total Size </div>
                                        <div> {torrent.TotalSize} </div>
                                    </div>
                                    <div className="col ml-5">
                                        <FileMenu
                                            setPlayerData={this.props.setPlayerData}
                                            torrent={torrent}/>
                                    </div>
                                </div>
                            </div>
                        )
                    })
                } 
                </div>
            </div>
        )
    }
}