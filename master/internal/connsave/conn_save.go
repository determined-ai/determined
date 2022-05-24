package connsave

import (
	"context"
	"net"
)

type connKey struct {}

func SaveConn(ctx context.Context, c net.Conn) context.Context {
	return context.WithValue(ctx, connKey{}, c)
}

func GetConn(ctx context.Context) net.Conn {
	return ctx.Value(connKey{}).(net.Conn)
}
