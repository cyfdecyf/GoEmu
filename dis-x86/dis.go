package dis

import (
	"io"
)

/*
Disassembler for x86.

For each instruction, the disassemble contains several passes:

1. Parse prefix
2. Parse opcode

This assembler only parses the instruction and put the information in the
DisContext. This is intended to make it useful for different purpose.
*/

/* Register order is the same with Table 3.1 Register Codes in Intel manual 2A
   This makes it easy to get operand for instructions with "+rb, +rw, +rd,
   +ro" opcode column. */
const (
	Eax = iota
	Ecx
	Edx
	Ebx
	Esp
	Ebp
	Esi
	Edi
)

type Instrucion struct {
	Prefix   int
	Opcode   int
	Mod      int
	Operand1 int
	Operand2 int
	Operand3 int
	Operand4 int

	Raw  [6]byte
	Size int
}

// Disassemble. Record information in each pass.
type DisContext struct {
	// Record position in the binary code
	binary io.ReaderAt
	offset int64

	Instrucion
}

func (dc *DisContext) NextInsn() {
	dc.Size = 0
	dc.parsePrefix()
	dc.parseOpcode()
}

func parseModRM(b byte) (mod, reg, rm int) {
	/*
	   ModRM byte bit format:

	       mmgggrrr

	   m: mod
	   g: reg
	   r: rm
	*/
	mod = int(b) >> 6 & 0x3
	reg = int(b) >> 3 & 0x7
	rm = int(b) & 7
	return
}

var buf1 = make([]byte, 1)

// Get the next byte in the instrucion stream
func (dc *DisContext) getNextByte() byte {
	_, err := dc.binary.ReadAt(buf1, dc.offset)
	if err != nil {
		panic(err)
	}

	dc.offset++
	dc.Raw[dc.Size] = buf1[0]
	dc.Size++
	return buf1[0]
}

// Put back the previously read byte
func (dc *DisContext) putNextByte() {
	dc.Size--
	dc.Raw[dc.Size] = 0
	dc.offset--
}
