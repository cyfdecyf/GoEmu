package dis

// Disassembler for x86

var (
	// All instruction prefixes. From Intel manual 2A, Section 2.1.1
	prefix = map[uint8]string {
		// Group 1
		// Lock and repeat prefixes
		0xF0: "LOCK",
		0xF2: "REPNE/REPNZ",
		0xF3: "REPE/REPZ",

		// Group 2
		// Segment override prefixes (use with any branch instruction is reserved)
		//0x2E: "CS", // Same with branch not taken hint
		0x36: "SS",
		//0x3E: "DS", // Same with branch taken hint
		0x26: "ES",
		0x64: "FS",
		0x65: "GS",
		// Branch hints (used only with Jcc instructions)
		0x2E: "Branch not taken",
		0x3E: "Branch taken",

		// Group 3
		0x66: "Operand-size override",

		// Group 4
		0x67: "Address-size override",
	}
)

