package provisioner

import (
	"sort"
	"strings"
)

// setOfSpotRequests is a light wrapper around a map to make it look like a set. Primarily
// exists to hide golang boilerplate behind descriptively named functions. Elements in the
// set are spotRequest pointers.
type setOfSpotRequests struct {
	keyMap map[string]*spotRequest
}

// add spotRequest to the set
func (c *setOfSpotRequests) add(s *spotRequest) *setOfSpotRequests {
	c.keyMap[s.SpotRequestID] = s
	return c
}

// delete spotRequest from the set
func (c *setOfSpotRequests) delete(s *spotRequest) *setOfSpotRequests {
	delete(c.keyMap, s.SpotRequestID)
	return c
}

// deleteByID deletes spotRequest with the given id from the set
func (c *setOfSpotRequests) deleteByID(s string) *setOfSpotRequests {
	delete(c.keyMap, s)
	return c
}

// deleteIntersection delete any items that exist in both set and set2 from set.
func (c *setOfSpotRequests) deleteIntersection(set2 setOfSpotRequests) *setOfSpotRequests {
	for _, req := range set2.iter() {
		c.deleteByID(req.SpotRequestID)
	}
	return c
}

// copy creates a shallow copy of the set
func (c *setOfSpotRequests) copy() setOfSpotRequests {
	set := newSetOfSpotRequests()
	for _, req := range c.iter() {
		set.add(req)
	}
	return set
}

// asList return the spotRequests in the set as a slice
func (c *setOfSpotRequests) asList() []*spotRequest {
	list := make([]*spotRequest, 0, len(c.keyMap))
	for _, sr := range c.keyMap {
		list = append(list, sr)
	}
	return list
}

// asListInChronologicalOrder returns the spotRequests in the set as a slice,
// sorted in chronological order
func (c *setOfSpotRequests) asListInChronologicalOrder() []*spotRequest {
	l := c.asList()
	sort.SliceStable(l, func(i, j int) bool {
		return l[i].CreationTime.Before(l[j].CreationTime)
	})
	return l
}

// instanceIds goes through the spotRequests and returns all instanceIds that are not nil
func (c *setOfSpotRequests) instanceIds() []*string {
	instanceIDs := make([]*string, 0)
	for _, req := range c.keyMap {
		if req.InstanceID != nil {
			b := strings.Builder{}
			b.WriteString(*req.InstanceID)
			i := b.String()
			instanceIDs = append(instanceIDs, &i)
		}
	}
	return instanceIDs
}

// idsAsList returns the spotRequest ids as a slice of strings
func (c *setOfSpotRequests) idsAsList() []string {
	list := make([]string, 0, len(c.keyMap))
	for reqID := range c.keyMap {
		list = append(list, reqID)
	}
	return list
}

// idsAsListOfPointers returns the spotRequest ids as a slice of string pointers
func (c *setOfSpotRequests) idsAsListOfPointers() []*string {
	list := make([]*string, 0, len(c.keyMap))
	for reqID := range c.keyMap {
		b := strings.Builder{}
		b.WriteString(reqID)
		k := b.String()
		list = append(list, &k)
	}
	return list
}

// iter returns the underlying map to make iterating over setOfSpotRequests as clean as
// iterating over a map. e.g.:
// for reqId, req := range set.iter() { ... }
func (c *setOfSpotRequests) iter() map[string]*spotRequest {
	return c.keyMap
}

// numReqs returns the number of spotRequests in the set
func (c *setOfSpotRequests) numReqs() int {
	return len(c.keyMap)
}

// newSetOfSpotRequests creates a new, empty setOfSpotRequests
func newSetOfSpotRequests() setOfSpotRequests {
	return setOfSpotRequests{
		keyMap: make(map[string]*spotRequest),
	}
}

// setOfStrings is a light wrapper around a map to make it look like a set. Primarily
// exists to hide golang boilerplate behind descriptively named functions. Elements
// in the set are strings.
type setOfStrings struct {
	keyMap map[string]bool
}

// add a string to the set
func (c *setOfStrings) add(s string) {
	c.keyMap[s] = true
}

// length returns the number of items in the set
func (c *setOfStrings) length() int {
	return len(c.keyMap)
}

// asList returns the strings in the set as a slice of strings
func (c *setOfStrings) asList() []string {
	list := make([]string, 0, len(c.keyMap))
	for key := range c.keyMap {
		list = append(list, key)
	}
	return list
}

// asListOfPointers returns the strings in the set as a slice of string pointers
func (c *setOfStrings) asListOfPointers() []*string {
	list := make([]*string, 0, len(c.keyMap))
	for key := range c.keyMap {
		b := strings.Builder{}
		b.WriteString(key)
		k := b.String()
		list = append(list, &k)
	}
	return list
}

// string returns a string representation of the set, which is the list of strings
// separated by commas
func (c *setOfStrings) string() string {
	l := c.asList()
	return strings.Join(l, ",")
}

// newSetOfStrings creates a new, empty setOfStrings
func newSetOfStrings() setOfStrings {
	return setOfStrings{
		keyMap: make(map[string]bool),
	}
}
