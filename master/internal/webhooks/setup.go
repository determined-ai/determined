package webhooks

import "context"

var singletonShipper *shipper

// Init creates a shipper singleton.
func Init(ctx context.Context) {
	singletonShipper = newShipper(ctx)
}

// Deinit closes a shipper.
func Deinit() {
	singletonShipper.Close()
}
