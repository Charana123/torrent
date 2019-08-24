package torrent

import (
	"fmt"
	"testing"
)

func TestNewTorrent(t *testing.T) {
	tfile := "/Users/charana/Downloads/08c2309bd3eaabf038b60ba8a82273fe8f474da2.torrent"
	tor, err := NewTorrent(tfile)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(tor.metaInfo.Info.PieceLength)
	fmt.Println(tor.metaInfo.Info.Files[0].Length)
	fmt.Println(tor.numPieces)
}
