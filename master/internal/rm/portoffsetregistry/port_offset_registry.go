package portoffsetregistry

import (
	"sync"

	"strconv"

	bst "github.com/gyuho/bst"
	"github.com/pkg/errors"
	orderedmap "github.com/wk8/go-ordered-map"
)

var (
	portOffsetRegistryOrderedMap *orderedmap.OrderedMap // or tree bst package in Go.
	portOffsetRegistryTree       *bst.Tree
	portOffsetRegistryMutex      sync.RWMutex
)

func NewPortOffsetRegistry() {
	// initialize Map with 0,range
	portOffsetRegistryOrderedMap = orderedmap.New()
	registryRange := 10000
	portOffsetRegistryOrderedMap.Set(0, registryRange)

	// initialize tree with node -1.
	root := bst.NewNode(bst.Int(-1))
	portOffsetRegistryTree = bst.New(root)
}

func GetPortOffset() (int, error) {
	portOffsetRegistryMutex.Lock()
	defer portOffsetRegistryMutex.Unlock()
	// Map implementation
	// Get lowest key in range 0 - 10,000
	//portOffsetRegistryOrderedMap[key+1] = val

	//Tree implementation
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
	portOffset := prevNum + 1 // lowest skipped number in registry or next value after the last in the registry.
	newNode := bst.NewNode(bst.Int(portOffset))
	portOffsetRegistryTree.Insert(newNode)
	return portOffset, nil
}

func RestorePortOffset(portOffset int) {
	portOffsetRegistryMutex.Lock()
	defer portOffsetRegistryMutex.Unlock()

	restoreNode := bst.NewNode(bst.Int(portOffset))
	portOffsetRegistryTree.Insert(restoreNode)

}

func ReleasePortOffset(portOffset bst.Int) bool {
	portOffsetRegistryMutex.Lock()
	defer portOffsetRegistryMutex.Unlock()

	// Map implementation
	/*
			prev_in_map = False
			if val - 1 in port_registry_map:
				prev_in_map = True
				if port_registry_map[val - 1] == null: // else don’t do anything
					port_registry_map[val - 1]  = val
			if val + 1 in port_registry_map:
		 		if prev_in_map:
					port_registry_map[val - 1]  = port_registry_map[val + 1] // update the end range here
				else:
					port_registry_map[val] =  port_registry_map[val + 1]
		        remove(val + 1) // remove this key, pair value from map.
		    else:
		        port_registry_map[val] = null
	*/

	// Tree implementation
	if portOffsetRegistryTree.Delete(portOffset) != nil {
		return true
	} else {
		return false
	}
}
