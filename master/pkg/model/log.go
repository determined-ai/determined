package model

// LogMessage is part of the output stream of a task. It is typically broken up
// into lines, separated by new-line characters ('\n'). Though if the output
// does not contain new-line characters, the message may be broken at arbitrary
// boundaries.
type LogMessage struct {
	ID      int       `db:"id" json:"id"`
	Message RawString `db:"message" json:"message"`
}
