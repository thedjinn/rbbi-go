package rbbi

// The Cursor is an interface representing an iterator over some kind of
// unicode rune backing store. This can be a string, a rune slice, or even a
// complex data structure such as a piece table or rope.
//
// The Cursor interface is quite minimal. Implementatinos only need to provide
// a position getter/setter and forward/backward iteration functions.
//
// A Cursor is stateful in that it has a current position. This position is
// represented as an int, but consumers of the Cursor should treat this value
// as being opaque, meaning that it can only be used for saving and restoring a
// previously encountered position.
//
// It is up to the implementer of the interface to guarantee that any position
// is a one to one mapping to a specific location in the unicode string.
type Cursor interface {
	// Return the current position of the Cursor. The actual position value
	// will be treated as an opaque value in that it will only be used for
	// saving/restoring a previously encountered position.
	Position() int

	// Set the current Cursor position to the provided value. The exact meaning
	// of the position value can be decided on by the implementer of the
	// interface. An error is returned when an invalid position is provided.
	SetPosition(position int) error

	// Return the rune at the current iterator position and advance the
	// iterator to the next rune. The return value of ok is false when Next()
	// is invoked while the iterator was at the end of the string. This
	// indicates that iteration is done and there are no more runes to
	// retrieve.
	Next() (r rune, ok bool)

	// Return the rune at the current iterator position and retreat the
	// iterator to the previous rune. The return value of ok is false when
	// Previous() is invoked while the iterator was at the beginning of the
	// string. This indicates that iteration is done and there are no more
	// runes to retrieve.
	Previous() (r rune, ok bool)
}
