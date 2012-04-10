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
	PrefixOperandSize

	// Group 4
	// Address override
	PrefixAddressSize
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
	0x66: PrefixOperandSize,
	// Group 4
	0x67: PrefixAddressSize,
}

// Read only one byte, store information in the Prefix field
func (dc *DisContext) __parsePrefix() (got bool) {
	pref, ok := Prefix[dc.nextByte()]
	if ok {
		got = true
		dc.Prefix |= pref
		switch pref {
		case PrefixOperandSize:
			dc.opSizeOverride = true
		case PrefixAddressSize:
			dc.addrSizeOverride = true
		}
	} else {
		dc.putByte()
	}
	return
}

func (dc *DisContext) parsePrefix() {
	// Keep parsing prefix until no one is find.
	for dc.__parsePrefix() {
	}
}
