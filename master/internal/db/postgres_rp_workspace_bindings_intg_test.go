package db

import "testing"

func TestAddAndRemoveBindings(t *testing.T) {
	// Test single insert/delete
	// Test bulk insert/delete
	return
}

func TestBindingFail(t *testing.T) {
	// Test add the same binding multiple times - should fail
	// Test add same binding among bulk transaction - should fail the entire transaction
	// Test add workspace that doesn't exist
	// Test add RP that doesn't exist
	return
}

func TestListWorkspacesBindingRP(t *testing.T) {
	// pretty straightforward
	// don't list bindings that are invalid
	// if RP is unbound, return nothing
	return
}

func TestListRPsBoundToWorkspace(t *testing.T) {
	// pretty straightforward
	// don't list binding that are invalid
	// return unbound pools too (we pull from config)
	return
}

func TestListAllBindings(t *testing.T) {
	// pretty straightforward
	// list ALL bindings, even invalid ones
	// make sure to return unbound pools too (we pull from config)
	return
}

func TestOverwriteBindings(t *testing.T) {
	// Test overwrite bindings
	// Test overwrite pool that's not bound to anything currently
	return
}

func TestOverwriteFail(t *testing.T) {
	// Test overwrite adding workspace that doesn't exist
	// Test overwrite pool that doesn't exist
	return
}

func TestRemoveInvalidBinding(t *testing.T) {
	// remove binding that doesn't exist
	// bulk remove bindings that don't exist
	return
}
