package webhooks

import "context"

var singletonShipper *shipper

func Init(ctx context.Context) {
	singletonShipper = newShipper(ctx)
}

func Deinit() {
	singletonShipper.Close()
}
