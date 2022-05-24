package connsave

import (
	"context"
	"net"
)

type connKey struct{}

// SaveConn saves net.Conn into the context for future retrieval.
func SaveConn(ctx context.Context, c net.Conn) context.Context {
	return context.WithValue(ctx, connKey{}, c)
}

// GetConn retrieves net.Conn from the context.
func GetConn(ctx context.Context) net.Conn {
	return ctx.Value(connKey{}).(net.Conn)
}
