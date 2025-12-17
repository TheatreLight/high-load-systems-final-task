package analytics

import (
	"math"
	"sync"
)

const DefaultZScoreThreshold = 2.0

// ZScoreDetector detects anomalies using z-score method
type ZScoreDetector struct {
	window    []float64
	size      int
	threshold float64
	mu        sync.RWMutex
}

// NewZScoreDetector creates a new z-score anomaly detector
func NewZScoreDetector(windowSize int, threshold float64) *ZScoreDetector {
	if windowSize <= 0 {
		windowSize = DefaultWindowSize
	}
	if threshold <= 0 {
		threshold = DefaultZScoreThreshold
	}
	return &ZScoreDetector{
		window:    make([]float64, 0, windowSize),
		size:      windowSize,
		threshold: threshold,
	}
}

// Add adds a value and returns whether it's an anomaly
func (zd *ZScoreDetector) Add(value float64) (isAnomaly bool, zscore float64) {
	zd.mu.Lock()
	defer zd.mu.Unlock()

	// Calculate z-score before adding the new value
	zscore = zd.calculateZScore(value)
	isAnomaly = math.Abs(zscore) > zd.threshold

	// Add to window
	if len(zd.window) >= zd.size {
		zd.window = zd.window[1:]
	}
	zd.window = append(zd.window, value)

	return isAnomaly, zscore
}

// calculateZScore computes the z-score for a value (must hold lock)
func (zd *ZScoreDetector) calculateZScore(value float64) float64 {
	if len(zd.window) < 2 {
		return 0
	}

	mean := zd.calculateMean()
	stddev := zd.calculateStdDev(mean)

	if stddev == 0 {
		return 0
	}

	return (value - mean) / stddev
}

// calculateMean computes the mean of window values (must hold lock)
func (zd *ZScoreDetector) calculateMean() float64 {
	if len(zd.window) == 0 {
		return 0
	}

	sum := 0.0
	for _, v := range zd.window {
		sum += v
	}
	return sum / float64(len(zd.window))
}

// calculateStdDev computes the standard deviation (must hold lock)
func (zd *ZScoreDetector) calculateStdDev(mean float64) float64 {
	if len(zd.window) < 2 {
		return 0
	}

	sumSquares := 0.0
	for _, v := range zd.window {
		diff := v - mean
		sumSquares += diff * diff
	}

	variance := sumSquares / float64(len(zd.window))
	return math.Sqrt(variance)
}

// GetStats returns current mean and standard deviation
func (zd *ZScoreDetector) GetStats() (mean, stddev float64) {
	zd.mu.RLock()
	defer zd.mu.RUnlock()

	mean = zd.calculateMean()
	stddev = zd.calculateStdDev(mean)
	return mean, stddev
}

// IsAnomaly checks if a value is an anomaly without adding it
func (zd *ZScoreDetector) IsAnomaly(value float64) (bool, float64) {
	zd.mu.RLock()
	defer zd.mu.RUnlock()

	zscore := zd.calculateZScore(value)
	return math.Abs(zscore) > zd.threshold, zscore
}

// Count returns the number of values in the window
func (zd *ZScoreDetector) Count() int {
	zd.mu.RLock()
	defer zd.mu.RUnlock()
	return len(zd.window)
}

// Threshold returns the configured threshold
func (zd *ZScoreDetector) Threshold() float64 {
	return zd.threshold
}

// Reset clears all values from the window
func (zd *ZScoreDetector) Reset() {
	zd.mu.Lock()
	defer zd.mu.Unlock()
	zd.window = make([]float64, 0, zd.size)
}
