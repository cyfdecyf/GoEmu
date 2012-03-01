package dis

import (
	"io"
	"os"
	"log"
	"encoding/binary"
)

/*
Disassembler for x86.

For each instruction, the disassemble contains several passes:

1. Parse prefix
2. Parse opcode

This assembler only parses the instruction and put the information in the
DisContext. This is intended to make it useful for different purpose.
*/

var debug = log.New(os.Stderr, "DEBUG ", log.Lshortfile)

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

const (
	Al = iota
	Cl
	Dl
	Bl
	Ah
	Ch
	Dh
	Bh
)

const (
	OpSizeByte = iota
	OpSizeWord
	OpSizeLong // Long = DoubleWord
	OpSizeQuad
)

type Instrucion struct {
	Prefix int
	Opcode int

	Mod byte
	Reg byte
	Rm  byte

	Scale byte
	Index byte
	Base  byte

	Displacement int32
	Imm          int32

	RawOpCode [3]byte
}

// Disassemble. Record information in each pass.
type DisContext struct {
	// Record position in the binary code
	binary io.ReaderAt
	offset int64

	Dflag     bool // Affects the operand-size and address-size attributes
	Protected bool // in Protected mode?

	Instrucion
}

// Create a new DisContext with protected mode on, dflag set.
func NewDisContext(binary io.ReaderAt) (dc *DisContext) {
	dc = new(DisContext)
	dc.binary = binary
	dc.offset = 0
	dc.Dflag = true
	dc.Protected = true
	return
}

func (dc *DisContext) NextInsn() {
	dc.parsePrefix()
	dc.parseOpcode()
}

/* Reading binary */

func (dc *DisContext) getBytes(buf []byte) {
	n, err := dc.binary.ReadAt(buf, dc.offset)
	if err != nil && err != os.EOF {
		panic(err)
	}
	dc.offset += int64(n)
}

var (
	bufbyte = make([]byte, 1)
	bufword = make([]byte, 2)
	buflong = make([]byte, 4)
)

// Get the next byte in the instrucion stream
func (dc *DisContext) nextByte() byte {
	dc.getBytes(bufbyte)
	return bufbyte[0]
}

func (dc *DisContext) nextWord() uint16 {
	dc.getBytes(bufword)
	return binary.LittleEndian.Uint16(bufword)
}

func (dc *DisContext) nextLong() uint32 {
	dc.getBytes(buflong)
	return binary.LittleEndian.Uint32(buflong)
}

// Put back the previously read byte
func (dc *DisContext) putByte() {
	dc.offset--
}

/* ModR/M and SIB byte parsing */

func parseBitField(b byte) (mod, reg, rm byte) {
	/*
	   ModRM byte bit format:
	       mmgggrrr
	   m: mod, g: reg, r: rm
	*/
	mod = b >> 6 & 0x3
	reg = b >> 3 & 0x7
	rm = b & 7
	return
}

func (dc *DisContext) parseModRM() {
	dc.Mod, dc.Reg, dc.Rm = parseBitField(dc.nextByte())
	// XXX Is the addressing mode os ModR/M byte affected by the address-size
	// attribute?
	size := dc.calcAddressSize(CalcOpSize)
	switch size {
	case OpSizeWord:
		dc.parseAfterModRM16bit()
	case OpSizeLong:
		dc.parseAfterModRM32bit()
	default:
		log.Fatalln("Address-size error")
	}
}

func (dc *DisContext) parseAfterModRM32bit() {
	// Refer to Intel Manual 2A Table 2-2
	switch dc.Mod {
	case 0:
		switch dc.Rm {
		case 4:
			dc.parseSIB()
		case 5:
			dc.Displacement = int32(dc.nextLong())
		}
	case 1:
		if dc.Rm == 4 {
			dc.parseSIB()
		}
		dc.Displacement = int32(dc.nextByte())
	case 2:
		if dc.Rm == 4 {
			dc.parseSIB()
		}
		dc.Displacement = int32(dc.nextLong())
	}
}

func (dc *DisContext) parseAfterModRM16bit() {
	// Refer to Intel Manual 2A Table 2-1
	switch dc.Mod {
	case 0:
		if dc.Rm == 6 {
			dc.Displacement = int32(dc.nextWord())
		}
	case 1:
		dc.Displacement = int32(dc.nextByte())
	case 2:
		dc.Displacement = int32(dc.nextWord())
	}
}

func (dc *DisContext) parseSIB() {
	// SIB has the same bit field allocation with ModR/M byte
	dc.Scale, dc.Index, dc.Base = parseBitField(dc.nextByte())
}

/* Displacement and immediate value */

// Convert byte to int. true = 1, false = 0
func Btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

const (
	CalcOpSize = true
	ByteOpSize = false
)

func (dc *DisContext) calcSize(calcSize bool, overridePrefix int) (size byte) {
	if calcSize {
		// At least word size, even for real mode
		size = OpSizeWord
		if dc.Protected {
			if dc.Prefix&overridePrefix != 0 {
				size += byte(Btoi(!dc.Dflag))
			} else {
				size += byte(Btoi(dc.Dflag))
			}
		}
	} else {
		size = OpSizeByte
	}
	return
}

func (dc *DisContext) calcOperandSize(calcSize bool) (size byte) {
	return dc.calcSize(calcSize, PrefixOprandSize)
}

func (dc *DisContext) calcAddressSize(calcSize bool) (size byte) {
	return dc.calcSize(calcSize, PrefixAddrSize)
}

func (dc *DisContext) getDisplacement(calcSize bool) (dis int32) {
	switch size := dc.calcOperandSize(calcSize); size {
	case OpSizeByte:
		dis = int32(dc.nextByte())
	case OpSizeWord:
		dis = int32(dc.nextWord())
	case OpSizeLong:
		dis = int32(dc.nextLong())
	default:
		log.Fatalf("displacement size error: %v\n", size)
	}
	return
}

func (dc *DisContext) getImmediate(calcsize bool) (dis int32) {
	return dc.getDisplacement(calcsize)
}
