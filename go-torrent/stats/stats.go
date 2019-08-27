package stats

import "sync"

type Stats interface {
	GetTrackerStats() (int, int, int)
}

type stats struct {
	sync.RWMutex
	uploaded   int
	downloaded int
	left       int
}

func NewStats() Stats {
	return &stats{}
}

func (s *stats) GetTrackerStats() (int, int, int) {
	s.RLock()
	defer s.RUnlock()
	return s.uploaded, s.downloaded, s.left
}
