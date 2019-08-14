package main

// import (
// 	"log"
// 	"net/http"
// )

// func main() {
// 	staticServeHandler := http.FileServer(http.Dir("/Users/charana/Documents/go/src/github.com/Charana123/torrent/public/"))
// 	http.Handle("/public/", http.StripPrefix("/public/", staticServeHandler))
// 	http.ListenAndServe(":8080", logRequest(http.DefaultServeMux))
// }

// func logRequest(next http.Handler) http.Handler {
// 	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
// 		log.Printf("%s %s %s\n", r.RemoteAddr, r.Method, r.URL)
// 		next.ServeHTTP(w, r)
// 	})
// }

import (
	"log"

	"github.com/Charana123/torrent/go-torrent"
)

func main() {
	// mi, err := torrent.NewMetaInfo("/Users/charana/Downloads/79EFD3085CFDF8C77189B9828D1C6A50659F863F.torrent")
	// if err != nil {
	// 	panic(err)
	// }

	// if err != nil {
	// 	panic(err)
	// }

	// peerIDBuffer := &bytes.Buffer{}
	// binary.Write(peerIDBuffer, binary.BigEndian, time.Now().Unix())
	// binary.Write(peerIDBuffer, binary.BigEndian, [12]byte{})
	// peerID := [20]byte{}
	// copy(peerID[:], peerIDBuffer.Bytes()[:20])

	// key := rand.Int31()

	// trackerResponse, err := torrent.GetPeers(
	// 	mi.AnnounceList,
	// 	&torrent.TrackerRequest{
	// 		InfoHash:   mi.InfoHash,
	// 		PeerID:     peerID,
	// 		Uploaded:   0,
	// 		Downloaded: 0,
	// 		Left:       1000,
	// 		Event:      "started",
	// 		Key:        key,
	// 	})
	// if err != nil {
	// }z
	// for _, peer := range trackerResponse.Peers {
	// 	fmt.Println(peer.Addr.String())
	// }

	// fmt.Println(metainfo.Announce)
	// fmt.Println(metainfo.CreationDate)
	// fmt.Println(metainfo.Comment)
	// fmt.Println(metainfo.CreatedBy)
	// fmt.Println(metainfo.Encoding)

	quit := make(chan int)
	server, serverPeerMChans, serverPort, err := torrent.NewServer(quit)
	if err != nil {
		log.Panicln()
	}
	go server.Serve()
	t, err := torrent.NewTorrent("/Users/charana/Downloads/08c2309bd3eaabf038b60ba8a82273fe8f474da2.torrent")
	if err != nil {
		log.Panicln(err)
	}
	go t.Start(serverPeerMChans, serverPort)
}

// func main() {
// 	// // source
// 	// stream := utils.NewStream([]string{"one", "two", "three"})

// 	// newStream := stream.Map(func(str string) int {
// 	// 	return len(str)
// 	// })

// 	// // destination
// 	// dest := make([]int, 0, 0)
// 	// newStream.Values(&dest)
// 	// fmt.Println(dest)

// func main() {
// 	// i := []interface{}{[]interface{}{1, 2, 3}, []interface{}{4, 5, 6}}
// 	// var ii interface{} = i
// 	// var j [][]int
// 	// utils.RecursiveAssert(&ii, &j)
// 	// fmt.Println(j)

// 	i := make(map[string]interface{})
// 	i["one"] = 1
// 	i["two"] = 2
// 	var ii interface{} = i
// 	j := make(map[string]int)
// 	utils.RecursiveAssert(&ii, &j)
// 	fmt.Println(j)
// }
