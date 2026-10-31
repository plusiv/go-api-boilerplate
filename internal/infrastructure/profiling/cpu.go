// Package profiling manages optional process profiles.
package profiling

import (
	"errors"
	"fmt"
	"os"
	"runtime/pprof"
)

// CPUProfile is an active CPU profile written to a file.
type CPUProfile struct{ file *os.File }

// StartCPU creates a profile file and begins recording CPU samples.
func StartCPU(path string) (*CPUProfile, error) {
	file, err := os.Create(path)
	if err != nil {
		return nil, fmt.Errorf("create CPU profile: %w", err)
	}
	if err := pprof.StartCPUProfile(file); err != nil {
		if closeErr := file.Close(); closeErr != nil {
			return nil, errors.Join(fmt.Errorf("start CPU profile: %w", err), fmt.Errorf("close CPU profile: %w", closeErr))
		}
		return nil, fmt.Errorf("start CPU profile: %w", err)
	}
	return &CPUProfile{file: file}, nil
}

// Close stops recording, flushes the profile, and closes its file.
func (p *CPUProfile) Close() error {
	pprof.StopCPUProfile()
	if err := p.file.Close(); err != nil {
		return fmt.Errorf("close CPU profile: %w", err)
	}
	return nil
}
