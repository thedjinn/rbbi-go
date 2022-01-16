package rbbi

type ucpTrieType uint8

const (
	ucpTrieTypeFast ucpTrieType = iota
	ucpTrieTypeSmall
)

type ucpTrieValueWidth uint8

const (
	// Note: the funny ordering here is identical to what is defined in
	// ucptrie.h
	ucpTrieValueWidth16 ucpTrieValueWidth = iota
	ucpTrieValueWidth32
	ucpTrieValueWidth8
)

type ucpTrie struct {
	trieType   ucpTrieType
	valueWidth ucpTrieValueWidth

	// TODO: Redundant?
	//indexLength uint16
	dataLength int32 // Note: was originally a uint32 but runes are int32

	index3NullOffset uint16 // TODO: Not used yet
	dataNullOffset   uint16 // TODO: Not used yet
	//shiftedHighStart uint16 // TODO: Redundant?

	highStart          int32  // Note: was originally a uint32 but runes are int32
	shifted12HighStart uint32 // TODO: Not used yet
	nullValueOffset    uint32 // TODO: Redundant? Only used in getRange

	index []uint16

	data8  []uint8
	data16 []uint16
	data32 []uint32

	nullValue uint32
}

// Internal constants
const (
	// Undocumented constants
	ucpTrieFastShift        int32 = 6
	ucpTrieSmallLimit       int32 = 0x1000
	ucpTrieSmallIndexLength int32 = ucpTrieSmallLimit >> ucpTrieFastShift

	// Number of entries in a data block for code points below the fast limit.
	// 64=0x40 @internal
	ucpTrieFastDataBlockLength int32 = 1 << ucpTrieFastShift

	// Mask for getting the lower bits for the in-fast-data-block offset.
	ucpTrieFastDataMask int32 = ucpTrieFastDataBlockLength - 1

	// Offset from dataLength (to be subtracted) for fetching the
	// value returned for code points highStart..U+10FFFF.
	ucpTrieHighValueNegDataOffset int32 = 2

	// Offset from dataLength (to be subtracted) for fetching the
	// value returned for out-of-range code points and ill-formed UTF-8/16.
	ucpTrieErrorValueNegDataOffset int32 = 1

	// The length of the BMP index table. 1024=0x400
	ucpTrieBmpIndexLength int32 = 0x10000 >> ucpTrieFastShift

	// Number of index-1 entries for the BMP. (4)
	// This part of the index-1 table is omitted from the serialized form.
	ucpTrieOmittedBmpIndex1Length int32 = 0x10000 >> ucpTrieShift1

	// Shift size for getting the index-3 table offset.
	ucpTrieShift3 int32 = 4

	// Shift size for getting the index-2 table offset.
	ucpTrieShift2 int32 = 5 + ucpTrieShift3

	// Shift size for getting the index-1 table offset.
	ucpTrieShift1 int32 = 5 + ucpTrieShift2

	// Difference between two shift sizes,
	// for getting an index-1 offset from an index-2 offset. 5=14-9
	ucpTrieShift1Minus2 int32 = ucpTrieShift1 - ucpTrieShift2

	// Difference between two shift sizes,
	// for getting an index-2 offset from an index-3 offset. 5=9-4
	ucpTrieShift2Minus3 int32 = ucpTrieShift2 - ucpTrieShift3

	// Number of entries in an index-2 block. 32=0x20
	ucpTrieIndex2BlockLength int32 = 1 << ucpTrieShift1Minus2

	// Number of entries in an index-3 block. 32=0x20
	ucpTrieIndex3BlockLength int32 = 1 << ucpTrieShift2Minus3

	// Mask for getting the lower bits for the in-index-2-block offset.
	ucpTrieIndex2Mask int32 = ucpTrieIndex2BlockLength - 1

	// Mask for getting the lower bits for the in-index-3-block offset.
	ucpTrieIndex3Mask int32 = ucpTrieIndex3BlockLength - 1

	// Number of entries in a small data block. 16=0x10
	ucpTrieSmallDataBlockLength int32 = 1 << ucpTrieShift3

	// Mask for getting the lower bits for the in-small-data-block offset.
	ucpTrieSmallDataMask int32 = ucpTrieSmallDataBlockLength - 1
)

// Undocumented internal function.
func (t *ucpTrie) internalSmallIndex(codePoint rune) int32 {
	var i1 int32 = codePoint >> ucpTrieShift1

	if t.trieType == ucpTrieTypeFast {
		if 0xffff >= codePoint || codePoint >= t.highStart {
			panic("Assertion error")
		}

		i1 += ucpTrieBmpIndexLength - ucpTrieOmittedBmpIndex1Length
	} else {
		if codePoint >= t.highStart || t.highStart <= ucpTrieSmallLimit {
			panic("Assertion error")
		}

		i1 += ucpTrieSmallIndexLength
	}

	var i3Block int32 = int32(t.index[int32(t.index[i1])+((codePoint>>ucpTrieShift2)&ucpTrieIndex2Mask)])
	var i3 int32 = (codePoint >> ucpTrieShift3) & ucpTrieIndex3Mask
	var dataBlock int32

	if (i3Block & 0x8000) == 0 {
		// 16-bit indexes
		dataBlock = int32(t.index[i3Block+i3])
	} else {
		// 18-bit indexes stored in groups of 9 entries per 8 indexes.
		i3Block = (i3Block & 0x7fff) + (i3 & ^7) + (i3 >> 3)
		i3 &= 7

		dataBlock = (int32(t.index[i3Block]) << (2 + (2 * i3))) & 0x30000
		i3Block++
		dataBlock |= int32(t.index[i3Block+i3])
	}

	return dataBlock + (codePoint & ucpTrieSmallDataMask)
}

// Internal trie getter for a code point below the fast limit. Returns the data
// index.
func (t *ucpTrie) fastIndex(codePoint rune) int32 {
	return int32(t.index[codePoint>>ucpTrieFastShift]) + (codePoint & ucpTrieFastDataMask)
}

// Internal trie getter for a code point at or above the fast limit. Returns
// the data index.
func (t *ucpTrie) smallIndex(codePoint rune) int32 {
	if int32(codePoint) >= t.highStart {
		return t.dataLength - ucpTrieHighValueNegDataOffset
	} else {
		return t.internalSmallIndex(codePoint)
	}
}

// Internal trie getter for a code point, with checking that codePoint is in
// U+0000..10FFFF.
func (t *ucpTrie) codePointIndex(fastMax int32, codePoint rune) int32 {
	if int32(codePoint) <= fastMax {
		return t.fastIndex(codePoint)
	} else {
		if codePoint <= 0x10ffff {
			return t.smallIndex(codePoint)
		} else {
			return t.dataLength - ucpTrieErrorValueNegDataOffset
		}
	}
}

// Returns a trie value for a code point, with range checking. Returns the trie
// error value if c is not in the range 0..U+10FFFF.
func (t *ucpTrie) fastGet(codePoint rune) uint32 {
	index := t.codePointIndex(0xffff, codePoint)

	switch t.valueWidth {
	case ucpTrieValueWidth8:
		return uint32(t.data8[index])
	case ucpTrieValueWidth16:
		return uint32(t.data16[index])
	case ucpTrieValueWidth32:
		return t.data32[index]
	}

	return 0
}
