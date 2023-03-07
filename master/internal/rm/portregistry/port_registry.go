package portregistry

import (
	"strconv"
	"sync"

	bst "github.com/gyuho/bst"
	"github.com/pkg/errors"
)

var (
	portRegistryTree  *bst.Tree
	portRegistryMutex sync.RWMutex
)

// NewPortRegistry initializes the global port registry tree.
func NewPortRegistry() {
	// initialize tree with node -1 because tree cannot be empty.
	root := bst.NewNode(bst.Int(-1))
	portRegistryTree = bst.New(root)
}

// GetPort returns available port above the given port base.
func GetPort(portBase int) (int, error) {
	portRegistryMutex.Lock()
	defer portRegistryMutex.Unlock()

	chInorder := make(chan string)
	go portRegistryTree.InOrder(chInorder)
	prevNum := portBase - 1 // we only care about ports after the port base
	for {
		v, ok := <-chInorder
		if !ok {
			break
		}
		vInt, err := strconv.Atoi(v)
		if err != nil {
			return -1, errors.Wrap(err, "getting port offsets from registry")
		}
		if vInt >= portBase && (vInt-1) != prevNum {
			break
		}
		if vInt >= portBase { // we want to ignore all nodes lesser than port base.
			prevNum = vInt
		}
	}
	port := prevNum + 1 // lowest skipped number in registry
	// or next value after the last in the registry (and above port base).
	newNode := bst.NewNode(bst.Int(port))
	portRegistryTree.Insert(newNode)
	return port, nil
}

// RestorePort restores a port by adding it back to the global port registry tree.
func RestorePort(port int) {
	portRegistryMutex.Lock()
	defer portRegistryMutex.Unlock()

	restoreNode := bst.NewNode(bst.Int(port))
	portRegistryTree.Insert(restoreNode)
}

// ReleasePort releases port and removes it from the port registry tree.
func ReleasePort(port int) bool {
	portRegistryMutex.Lock()
	defer portRegistryMutex.Unlock()

	if portRegistryTree.Delete(bst.Int(port)) != nil {
		return true
	}

	return false
}
