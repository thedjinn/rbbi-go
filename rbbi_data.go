package rbbi

type rbbiStateTableRow struct {
	accepting uint8
	lookahead uint8
	tagIndex  uint8

	// Note: length of nextStates is equal to RRBIData.categoryCount
	nextStates []uint8
}

type rbbiStateTableValueWidth uint8

const (
	rbbiStateTableValueWidth8 rbbiStateTableValueWidth = iota
	rbbiStateTableValueWidth16
)

type rbbiStateTable struct {
	stateCount           uint32 // TODO: same as len(rows)?
	rowLength            uint32
	dictCategoriesStart  uint32
	lookaheadResultsSize uint32

	// Flags
	lookaheadHardBreak bool                     // 0x1, legacy flag? Not used by ICU
	bofRequired        bool                     // 0x2
	valueWidth         rbbiStateTableValueWidth // 0x4 (to use 8 bits)

	// TODO: Tables can be either 8 or 16 bits
	rows []rbbiStateTableRow
}

type rbbiData struct {
	forwardTable rbbiStateTable
	reverseTable rbbiStateTable

	trie ucpTrie

	categoryCount uint32

	// TODO: Rule source?
	// TODO: Status table?

	// TODO: More stuff from the header
}
