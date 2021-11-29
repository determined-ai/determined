package actor

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"strings"
)

// Address is the location of an actor within an actor system.
type Address struct {
	path string
}

var rootAddress = Address{path: "/"}

// Addr returns a new address with the provided actor path components. Each of the path
// components must be URL-safe.
func Addr(rawPath ...interface{}) Address {
	if len(rawPath) == 0 {
		panic("must have a non-empty address")
	}
	path := make([]string, 0, len(rawPath))
	for _, rawPart := range rawPath {
		part := fmt.Sprint(rawPart)
		if strings.ContainsAny(part, "/") {
			panic("address path parts cannot contain a slash")
		}
		path = append(path, part)
	}
	parsed, err := url.Parse("/" + strings.Join(path, "/"))
	if err != nil {
		panic(err)
	}
	return Address{path: parsed.String()}
}

// AddrFromString is the inverse of `Address.String()`.
func AddrFromString(fullPath string) Address {
	return Address{path: fullPath}
}

func (a Address) String() string {
	return a.path
}

// Parent returns this actor's parent address.
func (a Address) Parent() Address {
	return Address{path: path.Dir(a.path)}
}

// Child returns a new address that is a child of this address.
func (a Address) Child(child interface{}) Address {
	id := fmt.Sprint(child)
	if strings.ContainsAny(id, "/") {
		panic("address path parts cannot contain a slash")
	}
	return Address{path: path.Join(a.path, id)}
}

// Local returns the local ID of the actor relative to the parent's ID space.
func (a Address) Local() string {
	return path.Base(a.path)
}

// IsAncestorOf returns true if the provided address is a descendant of this address.
func (a Address) IsAncestorOf(address Address) bool {
	if a == rootAddress {
		return a != address
	}
	return strings.HasPrefix(address.path, a.path+"/")
}

// nextParent returns the closest child address of the current address that is also an ancestor
// of the provided address. If the address is a direct child, the result is equal to the address.
// This is useful for traversing the actor hierarchy to find a descendant.
func (a Address) nextParent(address Address) Address {
	if !a.IsAncestorOf(address) {
		panic(fmt.Sprintf(
			"cannot fetch next parent for address: %v is not an ancestor of %v", a, address))
	}
	trimmed := address.path
	if a != rootAddress {
		trimmed = strings.TrimPrefix(address.path, a.path)
	}
	nextParentID := strings.SplitN(trimmed, "/", 3)[1]
	return a.Child(nextParentID)
}

// MarshalJSON implements the json.Marshaler interface.
func (a Address) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.path)
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (a *Address) UnmarshalJSON(data []byte) error {
	return json.Unmarshal(data, &a.path)
}

// MarshalText implements the encoding.TextMarshaler interface.
func (a Address) MarshalText() (text []byte, err error) {
	return []byte(a.path), nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface.
func (a *Address) UnmarshalText(text []byte) error {
	a.path = string(text)
	return nil
}
