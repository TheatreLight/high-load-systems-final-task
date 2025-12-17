package analytics

import (
	"sync"
)

const DefaultWindowSize = 50

// RollingAverage calculates rolling average over a sliding window
type RollingAverage struct {
	window []float64
	size   int
	mu     sync.RWMutex
}

// NewRollingAverage creates a new RollingAverage with specified window size
func NewRollingAverage(size int) *RollingAverage {
	if size <= 0 {
		size = DefaultWindowSize
	}
	return &RollingAverage{
		window: make([]float64, 0, size),
		size:   size,
	}
}

// Add adds a value to the window and returns the current average
func (ra *RollingAverage) Add(value float64) float64 {
	ra.mu.Lock()
	defer ra.mu.Unlock()

	// Remove oldest value if window is full
	if len(ra.window) >= ra.size {
		ra.window = ra.window[1:]
	}
	ra.window = append(ra.window, value)

	return ra.calculateAverage()
}

// calculateAverage computes the average of values in window (must hold lock)
func (ra *RollingAverage) calculateAverage() float64 {
	if len(ra.window) == 0 {
		return 0
	}

	sum := 0.0
	for _, v := range ra.window {
		sum += v
	}
	return sum / float64(len(ra.window))
}

// GetAverage returns the current rolling average
func (ra *RollingAverage) GetAverage() float64 {
	ra.mu.RLock()
	defer ra.mu.RUnlock()
	return ra.calculateAverage()
}

// GetValues returns a copy of current window values
func (ra *RollingAverage) GetValues() []float64 {
	ra.mu.RLock()
	defer ra.mu.RUnlock()

	result := make([]float64, len(ra.window))
	copy(result, ra.window)
	return result
}

// Count returns the number of values in the window
func (ra *RollingAverage) Count() int {
	ra.mu.RLock()
	defer ra.mu.RUnlock()
	return len(ra.window)
}

// WindowSize returns the configured window size
func (ra *RollingAverage) WindowSize() int {
	return ra.size
}

// Reset clears all values from the window
func (ra *RollingAverage) Reset() {
	ra.mu.Lock()
	defer ra.mu.Unlock()
	ra.window = make([]float64, 0, ra.size)
}

// GetPrediction returns the rolling average as a simple prediction
// The rolling average serves as a smoothed prediction of the next value
func (ra *RollingAverage) GetPrediction() float64 {
	return ra.GetAverage()
}
