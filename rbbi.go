// A Go port of ICU4C's Rule-Based Break Iterator (RBBI) algorithm for
// extracting various types of breaks (character/grapheme cluster, line,
// sentence, and word) from unicode strings.
package rbbi

// A struct representing the Unicode rule-based break iterator (RBBI). This
// struct encapsulates a state machine and lookup tables to detect various
// kinds of breaks in unicode strings.
//
// The break iterator takes a Cursor implementation, from which it scans runes.
// By using an interface for the Cursor the break iterator can be used with
// many different data structures, as long as they have the ability to yield
// runes and detect start/end of string boundaries.
type RBBI struct {
	data *rbbiData

	lookaheadMatches []int
	ruleStatusIndex  int32

	// Text that is iterated over
	cursor Cursor
}

func newRBBI(data *rbbiData) *RBBI {
	return &RBBI{
		data: data,

		lookaheadMatches: make([]int, data.forwardTable.lookaheadResultsSize),
	}
}

// Instantiate a new rule-based break iterator for detecting character
// (grapheme cluster) breaks.
func NewCharacterRBBI() *RBBI {
	return newRBBI(&rbbiCharacterData)
}

// Instantiate a new rule-based break iterator for detecting line breaks (for
// use with word wrapping).
func NewLineRBBI() *RBBI {
	return newRBBI(&rbbiLineData)
}

// Instantiate a new rule-based break iterator for detecting sentence braeks.
func NewSentenceRBBI() *RBBI {
	return newRBBI(&rbbiSentenceData)
}

// Instantiate a new rule-based break iterator for detecting word boundary
// breaks (e.g. for selecting a word by double-clicking).
func NewWordRBBI() *RBBI {
	return newRBBI(&rbbiWordData)
}

// Assign a new Cursor to the break iterator.
func (r *RBBI) SetCursor(cursor Cursor) {
	r.cursor = cursor

	// TODO: Invalidate break/dictionary caches
	// TODO: Call First()
}

const (
	// The state number of the starting state
	rbbiStateStart int32 = 1

	// The state-transition value indicating "stop"
	rbbiStateStop int32 = 0

	// Value constant for RBBIStateTableRow::fAccepting
	rbbiAcceptingUnconditional int16 = 1
)

type rbbiRunMode int

const (
	// State machine processing is before first char of input
	rbbiRunModeStart rbbiRunMode = iota

	// State machine processing is in the user text
	rbbiRunModeRun

	// State machine processing is after end of user text
	rbbiRunModeEnd
)

// Scan runes from the Cursor (in the forward direction) and stop at the next
// break. Returns a (position, ok) tuple after scanning. The value of ok is
// false when the iterator tried to scan beyond the end of the string. In any
// other case, ok is set to true and the first position immediately following
// the break is returned in the first return value. The Cursor is also updated
// to this position, allowing the Next() call to be executed as part of an
// iteration sequence.
//
// On failure the Cursor is reset to the position it had at the start of the
// Next() call.
func (r *RBBI) Next() (position int, ok bool) {
	var category uint16 = 0

	// handleNext always sets the break tag value.
	// Set the default for it.
	r.ruleStatusIndex = 0

	// TODO: Figure out what this is used for
	fDictionaryCharCount := 0

	initialPosition := r.cursor.Position()
	result := initialPosition

	// Grab the next rune
	c, nextOk := r.cursor.Next()

	// If we're already at the end of the text, return DONE.
	if !nextOk {
		return -1, false
	}

	// Set the initial state for the state machine
	state := rbbiStateStart
	row := r.data.forwardTable.rows[state]

	mode := rbbiRunModeRun
	if r.data.forwardTable.bofRequired {
		category = 2
		mode = rbbiRunModeStart
	}

	// Loop until we reach the end of the text or transition to state 0
	for {
		if !nextOk {
			// Reached end of input string.
			if mode == rbbiRunModeEnd {
				// We have already run the loop one last time with the
				// character set to the psueudo {eof} value. Now it is time to
				// unconditionally bail out.
				break
			}

			// Run the loop one last time with the fake end-of-input character
			// category.
			mode = rbbiRunModeEnd
			category = 1
		}

		// Get the char category. An incoming category of 1 or 2 means that we
		// are preset for doing the beginning or end of input, and that we
		// shouldn't get a category from an actual text input character.
		if mode == rbbiRunModeRun {
			// Look up the current character's character category, which tells
			// us which column in the state table to look at.
			category = uint16(r.data.trie.fastGet(c))

			if uint32(category) >= r.data.forwardTable.dictCategoriesStart {
				fDictionaryCharCount++
			}
		}

		// State Transition - move machine to its next state

		// fNextState is a variable-length array.
		if uint32(category) >= r.data.categoryCount {
			//U_ASSERT(category < fData->fHeader->fCatCount)
			panic("Assertion error")
		}

		state = int32(row.nextStates[category])
		row = r.data.forwardTable.rows[state]

		accepting := int16(row.accepting)
		if accepting == rbbiAcceptingUnconditional {
			// Match found, common case.
			if mode != rbbiRunModeStart {
				result = r.cursor.Position()
			}

			// Remember the break status (tag) values.
			r.ruleStatusIndex = int32(row.tagIndex)
		} else if accepting > rbbiAcceptingUnconditional {
			// Lookahead match is completed.
			if uint32(accepting) >= r.data.forwardTable.lookaheadResultsSize {
				//U_ASSERT(accepting < fData->fForwardTable->fLookAheadResultsSize);
				panic("Assertion error")
			}

			lookaheadResult := r.lookaheadMatches[accepting]

			if lookaheadResult >= 0 {
				r.ruleStatusIndex = int32(row.tagIndex)
				r.cursor.SetPosition(int(lookaheadResult))

				return int(lookaheadResult), true
			}
		}

		// If we are at the position of the '/' in a look-ahead (hard break)
		// rule; record the current position, to be returned later, if the full
		// rule matches.
		rule := row.lookahead

		if rule != 0 && int16(rule) <= rbbiAcceptingUnconditional {
			//U_ASSERT(rule == 0 || rule > ACCEPTING_UNCONDITIONAL);
			panic("Assertion failure")
		}

		if rule != 0 && uint32(rule) >= r.data.forwardTable.lookaheadResultsSize {
			//U_ASSERT(rule == 0 || rule < fData->fForwardTable->fLookAheadResultsSize);
			panic("Assertion failure")
		}

		if int16(rule) > rbbiAcceptingUnconditional {
			r.lookaheadMatches[rule] = r.cursor.Position()
		}

		if state == rbbiStateStop {
			// This is the normal exit from the lookup state machine. We have
			// advanced through the string until it is certain that no longer
			// match is possible, no matter what characters follow.
			break
		}

		// Advance to the next character. If this is a beginning-of-input loop
		// iteration, don't advance the input position. The next iteration will
		// be processing the first real input character.
		if mode == rbbiRunModeRun {
			c, nextOk = r.cursor.Next()
		} else {
			if mode == rbbiRunModeStart {
				mode = rbbiRunModeRun
			}
		}
	}

	// The state machine is done.  Check whether it found a match...

	// If the iterator failed to advance in the match engine, force it ahead by
	// one. (This really indicates a defect in the break rules. They should
	// always match at least one character.)
	if result == initialPosition {
		r.cursor.SetPosition(initialPosition)
		c, ok = r.cursor.Next()
		if !ok {
			return -1, false
		}

		result = r.cursor.Position()

		r.ruleStatusIndex = 0
	}

	// Leave the iterator at our result position.
	r.cursor.SetPosition(result)
	return result, true
}

// Iterate backwards using the safe reverse rules. The logic of this function
// is similar to Next(), but simpler because the safe table does not require as
// many options.
func (r *RBBI) safePrevious(fromPosition int) (position int, ok bool) {
	r.cursor.SetPosition(fromPosition)

	// Get the initial rune and bail out if we are already at the start of the
	// string.
	c, ok := r.cursor.Previous()
	if !ok {
		return -1, false
	}

	// Set the initial state for the state machine
	state := rbbiStateStart
	row := r.data.reverseTable.rows[state]

	// Loop until we reach the start of the text or transition to state 0
	for ok {
		// Look up the current character's character category, which tells us
		// which column in the state table to look at.
		category := r.data.trie.fastGet(c)

		if category >= r.data.categoryCount {
			//U_ASSERT(category<fData->fHeader->fCatCount);
			panic("Assertion error")
		}

		// State Transition - move machine to its next state
		state = int32(row.nextStates[category])
		row = r.data.reverseTable.rows[state]

		if state == rbbiStateStop {
			// This is the normal exit from the lookup state machine.
			// Transition to state zero means we have found a safe point.
			break
		}

		c, ok = r.cursor.Previous()
	}

	// The state machine is done. Check whether it found a match...
	result := r.cursor.Position()

	// TODO: When is ok false here?
	return result, true
}

// Scan runes from the Cursor (in the backward direction) and stop at the first
// break immediately preceding the break to the left of the cursor, or the
// first break preceding the Cursor if there is no break immediately left of
// it. Returns a (position, ok) tuple after scanning. The value of ok is false
// when the iterator tried to scan beyond the start of the string. In any other
// case, ok is set to true and the first position immediately following the
// break is returned in the first return value. The Cursor is also updated to
// this position, allowing the Previous() call to be executed as part of an
// iteration sequence.
//
// On failure the Cursor is reset to the position it had at the start of the
// Previous() call.
func (r *RBBI) Previous() (position int, ok bool) {
	// Save cursor position
	startPosition := r.cursor.Position()
	backtraceStart := startPosition

	// TODO: Take care of case when position = 0

	// Loop until we've found a last breakpoint that is not equal to
	// startPosition.
	lastBreakpoint := -1
	for lastBreakpoint == -1 {
		// Scan backwards for a safe point
		newStart, ok := r.safePrevious(backtraceStart)
		if !ok {
			if backtraceStart == startPosition {
				// Tried scanning before start of string when we were still at
				// the start position. This means that startPosition was the
				// beginning of the string. We can't go before this point, so
				// return false. There is no need to set the cursor, because
				// safePrevious did not change it.
				return -1, false
			}

			// Tried scanning before start of string, revert cursor to start
			// position and return the start of string position.
			if err := r.cursor.SetPosition(backtraceStart); err != nil {
				// We were already able to go to the start position. This should be
				// treated as an assertion error.
				panic("Assertion error")
			}

			return backtraceStart, true
		}

		// New safe point was not before start of string, save it so we can use
		// it as the origin when scanning for the next safe point.
		backtraceStart = newStart

		// Find last breakpoint before startPosition
		for {
			// Scan forward
			breakpoint, ok := r.Next()
			if !ok {
				// TODO: Can this happen when startPosition is the end of
				// the string? Need test case for this.
				//
				// Shouldn't happen because we know that startPosition is either at
				// the end of the string (when ok should be false) or before it.
				// Since we stop before we reach startPosition the value of ok
				// should always be true.
				panic("Assertion error")
			}

			// Break if we've reached startPosition
			if r.cursor.Position() >= startPosition {
				break
			}

			// If we reach here we've found a breakpoint that is not the same
			// as startPosition. Record it so we keep track of the last
			// occurrence of this event.
			lastBreakpoint = breakpoint
		}
	}

	// Set cursor to last breakpoint position (it is now at startPosition)
	// and return true.
	if err := r.cursor.SetPosition(lastBreakpoint); err != nil {
		// We were already able to go to this position. This should be
		// treated as an assertion error.
		panic("Assertion error")
	}

	// Return the breakpoint position
	return lastBreakpoint, true
}

// TODO: Next()
// TODO: Next(delta)
// TODO: Previous
// TODO: First
// TODO: Last
// TODO: Following(offset)
// TODO: Preceding(offset)
// TODO: IsBoundary(offset)
// TODO: Current

// TODO: GetRuleStatus
// TODO: GetRuleStatusVec (why?)

// TODO: HandleNext with state machine algorithm
// TODO: HandleSafePrevious with state machine
