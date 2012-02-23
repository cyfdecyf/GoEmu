package dis

// Instruction prefixes. From Intel manual 2A, Section 2.1.1
const (
	// Group 1
	// Lock and repeat prefixes
	prefixLOCK = 1 << iota
	prefixREPNZ
	prefixREPZ

	// Group 2
	// Segment override prefixes (use with any branch instruction is reserved)
	prefixCS
	prefixSS
	prefixDS
	prefixES
	prefixFS
	prefixGS
	// Branch hints:
	// Listed separately below

	// Group 3
	// Operand-size override
	prefixOPSIZE

	// Group 4
	// Address override
	prefixADDR
)

// Branch hints prefix, in group 2
const (
	prefixNotaken = prefixCS
	prefixTaken = prefixDS
)

var prefix = map[byte]int {
	// Group 1
	0xF0: prefixLOCK,
	0xF2: prefixREPNZ,
	0xF3: prefixREPZ,
	// Group 2
	0x2E: prefixCS,
	0x36: prefixSS,
	0x3E: prefixDS,
	0x26: prefixES,
	0x64: prefixFS,
	0x65: prefixGS,
	// Group 3
	0x66: prefixOPSIZE,
	// Group 4
	0x67: prefixADDR,
}

// Read only one byte, store information in the prefix field
func (dc *Disassembler) parsePrefix() {
	b := make([]byte, 1)
	_, err := dc.binary.ReadAt(b, dc.offset)
	if err != nil {
		panic(err)
	}

	pref, ok := prefix[b[0]]
	if ok {
		dc.prefix |= pref
		// Advance offset to disassbmle next byte
		dc.offset++
	}
}
