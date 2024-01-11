package archive

import "io"

type archiveClosers struct {
	closers []io.Closer
}

// Close() closes all items in closers in reverse order.
func (ac *archiveClosers) Close() error {
	for i := len(ac.closers) - 1; i >= 0; i-- {
		err := ac.closers[i].Close()
		if err != nil {
			return err
		}
	}
	return nil
}
