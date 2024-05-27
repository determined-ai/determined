package port

import (
	"sync"

	"github.com/pkg/errors"
)

// Range represents a range of ports.
type Range struct {
	Start     int
	End       int
	mu        sync.Mutex
	usedPorts map[int]bool
}

// Validates the structure.
func (p *Range) validate() error {
	// TODO: min range.
	if p.Start < 0 || p.End < 0 {
		return errors.Errorf("port range start and end must be positive")
	}
	if p.Start > p.End {
		return errors.Errorf("port range start must be less than or equal to port range end")
	}
	if p.End > 65535 {
		return errors.Errorf("port range end must be less than or equal to 65535")
	}
	for port := range p.usedPorts {
		if port < p.Start || port > p.End {
			return errors.Errorf("used port %d is not within the range", port)
		}
	}
	return nil
}

// LoadInUsedPorts loads the used ports into the range without raising for collisions.
func (p *Range) LoadInUsedPorts(usedPorts []int, ignoreOutOfRange bool) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, port := range usedPorts {
		if !p.contains(port) {
			if ignoreOutOfRange {
				continue
			}
			return errors.Errorf("port %d is not within the range", port)
		}
		p.usedPorts[port] = true
	}
	return nil
}

// Checks if a port is within the range.
func (p *Range) contains(port int) bool {
	return port >= p.Start && port <= p.End
}

// Finds the next available port that is not in the usedPorts list.
func (p *Range) nextAvailablePort() (int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for port := p.Start; port <= p.End; port++ {
		if !p.usedPorts[port] {
			p.usedPorts[port] = true
			return port, nil
		}
	}
	return 0, errors.Errorf("no available ports in the range")
}

// GetAndMarkUsed allocates and marks the specified number of ports as used.
func (p *Range) GetAndMarkUsed(count int) ([]int, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	allocatedPorts := []int{}
	for port := p.Start; port <= p.End && len(allocatedPorts) < count; port++ {
		if !p.usedPorts[port] {
			p.usedPorts[port] = true
			allocatedPorts = append(allocatedPorts, port)
		}
	}
	if len(allocatedPorts) < count {
		// Free the allocated ports if we couldn't allocate enough
		for _, port := range allocatedPorts {
			delete(p.usedPorts, port)
		}
		return nil, errors.Errorf("not enough available ports in the range")
	}
	return allocatedPorts, nil
}

// MarkPortAsUsed marks a port as used.
func (p *Range) MarkPortAsUsed(port int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.contains(port) {
		return errors.Errorf("port %d is not within the range", port)
	}
	if p.usedPorts[port] {
		return errors.Errorf("port %d is already used", port)
	}
	p.usedPorts[port] = true
	return nil
}

// MarkPortAsFree marks a port as free.
func (p *Range) MarkPortAsFree(port int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if !p.contains(port) {
		return errors.Errorf("port %d is not within the range", port)
	}
	if !p.usedPorts[port] {
		return errors.Errorf("port %d is not currently used", port)
	}
	delete(p.usedPorts, port)
	return nil
}

// NewRange creates a new port range with inclusive start and end ports.
func NewRange(start, end int, usedPorts []int) (*Range, error) {
	ports := make(map[int]bool)
	r := &Range{
		Start:     start,
		End:       end,
		mu:        sync.Mutex{},
		usedPorts: ports,
	}
	if err := r.validate(); err != nil {
		return nil, err
	}
	if err := r.LoadInUsedPorts(usedPorts, true); err != nil {
		return nil, err
	}
	return r, nil
}
