package client

import "io"

type FileDownload interface {
	Length() int
	NewReader() io.ReadSeeker
	Path() string
	Name() string
	PercentageComplete() float32
}

type fileDownload struct {
	stopped bool
}
