package portoffsetregistry

import (
	"strconv"
	"sync"

	bst "github.com/gyuho/bst"
	"github.com/pkg/errors"
)

var (
	portOffsetRegistryTree  *bst.Tree
	portOffsetRegistryMutex sync.RWMutex
)

// NewPortOffsetRegistry initializes the global port registry tree.
func NewPortOffsetRegistry() {
	// initialize tree with node -1.
	root := bst.NewNode(bst.Int(-1))
	portOffsetRegistryTree = bst.New(root)
}

// GetPortOffset returns available port offset.
func GetPortOffset() (int, error) {
	portOffsetRegistryMutex.Lock()
	defer portOffsetRegistryMutex.Unlock()

	chInorder := make(chan string)
	go portOffsetRegistryTree.InOrder(chInorder)
	prevNum := -2 // smallest number is always -1 because tree is initialized with root node as -1.
	for {
		v, ok := <-chInorder
		if !ok {
			break
		}
		vInt, err := strconv.Atoi(v)
		if err != nil {
			return -1, errors.Wrap(err, "getting port offsets from registry")
		}
		if (vInt - 1) != prevNum {
			break
		}
		prevNum = vInt
	}
	portOffset := prevNum + 1
	// lowest skipped number in registry or next value after the last in the registry.
	newNode := bst.NewNode(bst.Int(portOffset))
	portOffsetRegistryTree.Insert(newNode)
	return portOffset, nil
}

// RestorePortOffset restores a port offset by adding it back to the global port registry tree.
func RestorePortOffset(portOffset int) {
	portOffsetRegistryMutex.Lock()
	defer portOffsetRegistryMutex.Unlock()

	restoreNode := bst.NewNode(bst.Int(portOffset))
	portOffsetRegistryTree.Insert(restoreNode)
}

// ReleasePortOffset release port offset and removes it from the port registry tree.
func ReleasePortOffset(portOffset bst.Int) bool {
	portOffsetRegistryMutex.Lock()
	defer portOffsetRegistryMutex.Unlock()

	if portOffsetRegistryTree.Delete(portOffset) != nil {
		return true
	}

	return false
}
