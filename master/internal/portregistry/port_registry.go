package portregistry

import (
	"sync"

	rbt "github.com/emirpasic/gods/trees/redblacktree"
)

var (
	portRegistryTree  *rbt.Tree
	portRegistryMutex sync.RWMutex
)

// InitPortRegistry initializes the global port registry tree.
func InitPortRegistry(reservedPorts []int) {
	// initialize tree with node -1 because tree cannot be empty.
	portRegistryTree = rbt.NewWithIntComparator()
	registerAnyReservedPorts(reservedPorts)
}

func registerAnyReservedPorts(reservedPorts []int) {
	for _, port := range reservedPorts {
		portRegistryTree.Put(port, struct{}{}) // we only care about key.
	}
}

// IteratorAt(baseport node) or return base port

// GetPort returns available port above the given port base.
func GetPort(portBase int) (int, error) {
	portRegistryMutex.Lock()
	defer portRegistryMutex.Unlock()

	node := portRegistryTree.GetNode(portBase)
	if node == nil {
		portRegistryTree.Put(portBase, struct{}{}) // we only care about key.
		return portBase, nil
	}

	prevNum := portBase // we only care about ports here after the port base
	for it := portRegistryTree.IteratorAt(node); it.Next(); {
		v := it.Key().(int)
		if (v - 1) != prevNum {
			break
		}
		prevNum = v
	}
	port := prevNum + 1 // lowest skipped number in registry
	// or next value after the last in the registry (and above port base).
	portRegistryTree.Put(port, struct{}{}) // we only care about key.
	return port, nil
}

// RestorePort restores a port by adding it back to the global port registry tree.
func RestorePort(port int) {
	portRegistryMutex.Lock()
	defer portRegistryMutex.Unlock()

	portRegistryTree.Put(port, struct{}{})
}

// ReleasePort releases port and removes it from the port registry tree.
func ReleasePort(port int) {
	portRegistryMutex.Lock()
	defer portRegistryMutex.Unlock()

	portRegistryTree.Remove(port)
}
