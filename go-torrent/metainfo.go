package torrent

import (
	"crypto/sha1"
	"fmt"
	"os"
	"strings"

	utils "github.com/Charana123/go-utils"
	"github.com/marksamman/bencode"
)

type MetaInfo struct {
	Info         Info
	InfoHash     [20]byte
	Announce     string
	AnnounceList []string
	CreationDate int    // in seconds since epoch
	Comment      string // textual comment from the author
	CreatedBy    string // author of torrent
	Encoding     string // string encoding for info field
}

type Info struct {
	Name        string // single file - file name, multi file - parent directory
	Files       []File // single file - length of files is one
	PieceLength int64  // Bytes per piece (for all pieces)
	Pieces      string // Byte array for each 20-byte SHA-1 hash corresponding to a piece
	Private     bool   // (optional) Whether the torrent is private to the swarm managed by the listed trackers
}

type File struct {
	Length int64  // length of file in bytes
	MD5sum string // (optinal) MD5 sum, a 32-character hexadecimal string
	Path   string // File path
}

func NewMetaInfo(filePath string) (*MetaInfo, error) {
	file, err := os.Open(filePath)
	utils.HandleFatalError(nil, err)
	if err != nil {
		return nil, err
	}

	data, err := bencode.Decode(file)
	if err != nil {
		return nil, err
	}

	mi := &MetaInfo{}
	if announceList, ok := data["announce-list"]; ok {
		var announceListList [][]string
		utils.RecursiveAssert(&announceList, &announceListList)
		for _, al := range announceListList {
			mi.AnnounceList = append(mi.AnnounceList, al[0])
		}
		fmt.Println(mi.AnnounceList)
	} else {
		announce, _ := data["announce"]
		utils.RecursiveAssert(&announce, &mi.Announce)
	}

	if cd, ok := data["creation data"]; ok {
		utils.RecursiveAssert(&cd, &mi.CreationDate)
	}
	if c, ok := data["comment"]; ok {
		utils.RecursiveAssert(&c, &mi.Comment)
	}
	if cb, ok := data["created by"]; ok {
		utils.RecursiveAssert(&cb, &mi.CreatedBy)
	}
	if e, ok := data["encoding"]; ok {
		utils.RecursiveAssert(&e, &mi.Encoding)
	}

	if info, ok := data["info"]; ok {
		mi.InfoHash = sha1.Sum(bencode.Encode(info))
		var infoMap map[string]interface{}
		utils.RecursiveAssert(&info, &infoMap)

		if name, ok := infoMap["name"]; ok {
			utils.RecursiveAssert(&name, &mi.Info.Name)
		}

		if pl, ok := infoMap["piece length"]; ok {
			utils.RecursiveAssert(&pl, &mi.Info.PieceLength)
		}

		if ps, ok := infoMap["pieces"]; ok {
			utils.RecursiveAssert(&ps, &mi.Info.Pieces)
		}

		if p, ok := infoMap["private"]; ok {
			utils.RecursiveAssert(&p, &mi.Info.Private)
		}

		if files, ok := infoMap["files"]; ok {
			// multiple file mode
			var filesList []map[string]interface{}
			utils.RecursiveAssert(&files, &filesList)

			for _, fileMap := range filesList {
				file := File{}
				if length, ok := fileMap["length"]; ok {
					utils.RecursiveAssert(&length, &file.Length)
				}
				if md5sum, ok := fileMap["md5sum"]; ok {
					utils.RecursiveAssert(&md5sum, &file.MD5sum)
				}
				if path, ok := fileMap["path"]; ok {
					pathList := make([]string, 0, 0)
					utils.RecursiveAssert(&path, &pathList)
					file.Path = strings.Join(pathList, "")
				}
				mi.Info.Files = append(mi.Info.Files, file)
			}
		} else {
			// single file mode
			file := File{}
			if length, ok := infoMap["length"]; ok {
				utils.RecursiveAssert(&length, &file.Length)
			}
			if md5sum, ok := infoMap["md5sum"]; ok {
				utils.RecursiveAssert(&md5sum, &file.MD5sum)
			}
			mi.Info.Files = append(mi.Info.Files, file)
		}

	}

	return mi, nil
}
