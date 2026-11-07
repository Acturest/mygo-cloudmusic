package player

import "testing"

// The spectrum keeps the chunks near the one playing, not a song's all: an
// hour of an episode measured end to end would take megabytes for frames
// gone by the time they play.
func TestSpectrumKeepsNearbyChunks(t *testing.T) {
	s := newSpectrum(44100)
	s.acc = make([]float32, fftSize)
	last := int64(specKept + 100)
	for i := int64(0); i <= last; i++ {
		s.analyze(i)
	}
	if _, ok := s.at(last * fftSize); !ok {
		t.Fatalf("the chunk measured last is not there")
	}
	if _, ok := s.at((last - specKept) * fftSize); ok {
		t.Errorf("chunk %d kept, %d chunks back", last-specKept, specKept)
	}
	if _, ok := s.at((last + 1) * fftSize); ok {
		t.Errorf("a chunk never measured is there")
	}
	// A seek back past the window measures nothing until it plays on.
	if _, ok := s.at(0); ok {
		t.Errorf("the first chunk is still there")
	}
}
