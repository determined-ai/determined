package events

import "context"

// Publisher defines an interface on which the Docker lib publishes asynchronous events, such as
// logs or stats.
type Publisher[T any] interface {
	Publish(context.Context, T) error
}

// FuncPublisher wraps a plain func as a Publisher.
type FuncPublisher[T any] func(context.Context, T) error

// Publish implements Publisher for FuncPublisher.
func (f FuncPublisher[T]) Publish(ctx context.Context, e T) error {
	return f(ctx, e)
}

// NilPublisher is a publisher than does nothing and returns no errors. Useful for testing.
type NilPublisher[T any] struct{}

// Publish implements Publisher for NilPublisher.
func (f NilPublisher[T]) Publish(ctx context.Context, e T) error {
	return nil
}

// ChannelPublisher wraps a plain channel as a Publisher.
func ChannelPublisher[T any](events chan<- T) FuncPublisher[T] {
	return func(ctx context.Context, e T) error {
		select {
		case events <- e:
			return nil
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
