package dis

// Instruction Prefixes. From Intel manual 2A, Section 2.1.1
const (
	// Group 1
	// Lock and repeat Prefixes
	PrefixLOCK = 1 << iota
	PrefixREPNZ
	PrefixREPZ

	// Group 2
	// Segment override Prefixes (use with any branch instruction is reserved)
	PrefixCS
	PrefixSS
	PrefixDS
	PrefixES
	PrefixFS
	PrefixGS
	// Branch hints:
	// Listed separately below

	// Group 3
	// Operand-size override
	PrefixOPSIZE

	// Group 4
	// Address override
	PrefixADDR
)

// Branch hints Prefix, in group 2
const (
	PrefixNotaken = PrefixCS
	PrefixTaken   = PrefixDS
)

var Prefix = map[byte]int{
	// Group 1
	0xf0: PrefixLOCK,
	0xf2: PrefixREPNZ,
	0xf3: PrefixREPZ,
	// Group 2
	0x2e: PrefixCS,
	0x36: PrefixSS,
	0x3e: PrefixDS,
	0x26: PrefixES,
	0x64: PrefixFS,
	0x65: PrefixGS,
	// Group 3
	0x66: PrefixOPSIZE,
	// Group 4
	0x67: PrefixADDR,
}

// Read only one byte, store information in the Prefix field
func (dc *DisContext) parsePrefix() {
	pref, ok := Prefix[dc.getNextByte()]
	if ok {
		dc.Prefix |= pref
	} else {
		dc.putNextByte()
	}
}
