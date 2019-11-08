import React from 'react'

export default class Menu extends React.Component {
    constructor(props){
        super(props)
        this.state = {
            torrents: [
                {
                    Name: "torrent1",
                    infoHashHexString: "1",
                    Files: [
                        { Name: "file1" },
                        { Name: "file2" }
                    ],
                },
                {
                    Name: "torrent2",
                    infoHashHexString: "2",
                    Files: [
                        { Name: "file3" },
                        { Name: "file4" }
                    ]
                }
            ]
        }
    }

    render(){
        return (
            <div> { 
                this.state.torrents.map(torrent => {
                    return (
                        <div key={torrent.Name}>
                            <div> { torrent.Name } </div>
                            { torrent.Files.map((file, i)=> {
                                return (
                                    <div key={file.Name} 
                                        onClick={(event) => this.props.setPlayerData(torrent, i)}> 
                                        {file.Name} 
                                    </div>
                                )
                            }) }
                        </div>
                    )
                })
            } </div>
        )
    }
}