package rbbi

import (
	"errors"
	"unicode/utf8"
)

// The StringCursor is a Cursor implementation using a regular Go string as its
// backing store. The position values it uses are byte positions in the string.
type StringCursor struct {
	text     string
	position int
}

// Instantiate a new StringCursor using the provided string. The position of
// the cursor is initialized to be at the start of the string.
func NewStringCursor(text string) *StringCursor {
	return &StringCursor{
		text:     text,
		position: 0,
	}
}

// Return the current position of the StringCursor, represented as a byte
// offset relative to the start of the string.
func (c *StringCursor) Position() int {
	return c.position
}

// Set the current StringCursor position to the provided value. The new
// position should be stated as a byte offset relative to the start of the
// string. The position may not be a byte offset that is intersecting the bytes
// of a single rune. An error is returned when the provided position is outside
// the string's boundaries. A value equal to the string's byte size is legal
// and represents the end of the string.
func (c *StringCursor) SetPosition(position int) error {
	// Negative positions are invalid
	if position < 0 {
		return errors.New("Position can not be negative")
	}

	// Setting the position to len(c.text) is valid (this is the end of string
	// position), but setting it beyond is an error.
	if position > len(c.text) {
		return errors.New("Position can not be beyond the end of the string")
	}

	c.position = position
	return nil
}

// Return the rune at the current iterator position and advance the iterator to
// the next rune. The return value of ok is false when Next() is invoked while
// the iterator was at the end of the string. This indicates that iteration is
// done and there are no more runes to retrieve.
func (c *StringCursor) Next() (r rune, ok bool) {
	if c.position >= len(c.text) {
		return -1, false
	}

	r, size := utf8.DecodeRuneInString(c.text[c.position:])
	c.position += size

	return r, true
}

// Return the rune at the current iterator position and retreat the iterator to
// the previous rune. The return value of ok is false when Previous() is
// invoked while the iterator was at the beginning of the string. This
// indicates that iteration is done and there are no more runes to retrieve.
func (c *StringCursor) Previous() (r rune, ok bool) {
	if c.position <= 0 {
		return -1, false
	}

	r, size := utf8.DecodeLastRuneInString(c.text[:c.position])
	c.position -= size

	return r, true
}
