package disk

import (
	"os"
	"testing"

	"github.com/Charana123/torrent/go-torrent/torrent"
	"github.com/spf13/afero"
	"github.com/stretchr/testify/mock"
)

var mi *torrent.MetaInfo = &torrent.MetaInfo{
	Info: torrent.Info{
		PieceLength: 256, // 2^8
		Name:        "root",
		Files: []torrent.File{
			torrent.File{
				Length: 300,
				Path:   []string{"sub1", "name1"},
			},
			torrent.File{
				Length: 300,
				Path:   []string{"sub1", "sub2", "name2"},
			},
		},
	}}

func TestInit(t *testing.T) {
	// appFS = afero.NewMemMapFs()
	// disk := newDisk(mi)
	// disk.init()

	// if _, err := appFS.Stat("root"); os.IsNotExist(err) {
	// 	t.Error(err)
	// }
	// if _, err := appFS.Stat("root/sub1/name1"); os.IsNotExist(err) {
	// 	t.Error(err)
	// }
	// if _, err := appFS.Stat("root/sub1/sub2/name2"); os.IsNotExist(err) {
	// 	t.Error(err)
	// }
}

type mockFile struct {
	mock.Mock
	afero.File
}

func (m *mockFile) WriteAt(b []byte, off int64) (int, error) {
	args := m.Called(b, off)
	return args.Int(0), args.Error(1)
}

func (m *mockFile) ReadAt(b []byte, off int64) (int, error) {
	args := m.Called(b, off)
	return args.Int(0), args.Error(1)
}

func mockOpenFile(name string, flag int, perm os.FileMode) (afero.File, error) {
	return &mockFile{}, nil
}

func TestBlockReadRequest(t *testing.T) {
	// appFS = afero.NewMemMapFs()
	// openFile = mockOpenFile

	// disk := newDisk(mi)
	// disk.init()

	// // read 0th-index piece at offset 281, length 19
	// mf1 := disk.files[0].(*mockFile)
	// mf1.On("ReadAt", mock.MatchedBy(func(buf []byte) bool {
	// 	return len(buf) == 19
	// }), int64(281)).Return(19, nil)
	// // read 1th-index piece at offset 0, length 109
	// mf2 := disk.files[1].(*mockFile)
	// mf2.On("ReadAt", mock.MatchedBy(func(buf []byte) bool {
	// 	return len(buf) == 109
	// }), int64(0)).Return(109, nil)

	// breq := &blockReadRequest{
	// 	pieceIndex:      1,
	// 	blockByteOffset: 25,
	// 	length:          128, // 2^7
	// }

	// brespC := make(chan *blockReadResponse)
	// disk.BlockReadRequest(breq, brespC)

	// select {
	// case <-brespC:
	// 	mf1.AssertExpectations(t)
	// 	mf2.AssertExpectations(t)
	// case <-time.After(time.Second):
	// 	t.Error("timeout")
	// }
}

func TestWritePieceRequest(t *testing.T) {
	// appFS = afero.NewMemMapFs()
	// openFile = mockOpenFile

	// disk := newDisk(mi)
	// disk.init()

	// // read 0th-index piece at offset 281, length 19
	// mf1 := disk.files[0].(*mockFile) // 256
	// mf1.On("WriteAt", mock.MatchedBy(func(buf []byte) bool {
	// 	return len(buf) == 44
	// }), int64(256)).Return(44, nil)
	// // read 1th-index piece at offset 0, length 109
	// mf2 := disk.files[1].(*mockFile)
	// mf2.On("WriteAt", mock.MatchedBy(func(buf []byte) bool {
	// 	return len(buf) == 212
	// }), int64(0)).Return(212, nil)

	// preq := &pieceWriteRequest{
	// 	pieceIndex: 1,
	// 	data:       make([]byte, 256), // 2^8
	// }
	// disk.WritePieceRequest(preq)
	// <-time.After(time.Second)
	// mf1.AssertExpectations(t)
	// mf2.AssertExpectations(t)
}
