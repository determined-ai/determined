package aproto

const (
	// FluentImage is docker image id to use for the managed Fluent Bit daemon. Fluent Bit did a
	// rewrite of their TLS stack that introduced a bug (unconfirmed with them) with `tls.vhost`
	// in 1.7, so take care before upgrading this past that (1.9.3 did not work).
	FluentImage = "fluent/fluent-bit:1.6"
)
