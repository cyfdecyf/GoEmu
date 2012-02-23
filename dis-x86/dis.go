package dis

import (
	"io"
)

/*
Disassembler for x86.

For each instruction, the disassemble contains several passes:

1. Parse prefix
2. Parse opcode
*/

// Disassemble. Record information in each pass.
type Disassembler struct {
	// Record position in the binary code
	binary io.ReaderAt
	offset int64

	prefix int
}
