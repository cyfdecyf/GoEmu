package dis

import (
	"encoding/binary"
	"io"
	"log"
	"os"
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

/* Register order is the same with Table B-2 of Section B.1.4.2 in Vol 2C.
   This makes it easy to get operand for instructions with "+rb, +rw, +rd,
   +ro" opcode column. */
const (
	Eax byte = iota
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

/* Order confirms to Table B-8 in Intel Manual 2C. */
const (
	ES byte = iota
	CS
	SS
	DS
	FS
	GS
)

const (
	OpSizeCalc byte = iota // zero, means need to calculate the size or none
	OpSizeByte             // size starts from 1, so 0 can be used for none
	OpSizeWord
	OpSizeLong // Long = DoubleWord
	OpSizeQuad
)
const OpSizeNone byte = 0

// For operand type that does not have size suffix, it means the size depends
// on operand-size attribute.
// The same operand type with different suffix should be ordered.
const (
	OperandReg byte = iota
	OperandRegByte
	OperandRm
	OperandImm
	OperandImmByte
	// The size of the memory offset is determined by the address-size
	// attribute. The operand type for memory offset here specifies the size of
	// the data.
	OperandMOff
	OperandMOffByte
	OperandSegReg = 10 // Segment register
)

type InsnInfo struct {
	Opcode  byte
	Flag    uint64
	Operand [4]byte
}

type Instruction struct {
	Prefix int
	Opcode int

	Disp   int32 // Displacement
	ImmOff int32 // Immediate value or Offset

	Mod byte
	Reg byte
	Rm  byte

	Scale byte
	Index byte
	Base  byte

	Src      byte // source operand type
	Dst      byte // destination operand type
	Noperand byte // how many operands

	// Instruction specific operand/address size attribute.
	// This will only be set and overrides the information in DisContext if:
	// 1. The instruction has address/operand-size override prefix
	// 2. Or the instruction itself specifies these information
	//
	// To save space (as this is use in frequently), the high 4 bits specify
	// the address size, low 4 bits specify operand size.
	//
	// For emulation, if the instruction has size override prefix, the actual
	// size should always be calculated according to the current protected
	// mode and dflag. The disassembler can rely on this because it has no
	// dynamic information about the CPU.
	sizeOverride byte

	// Displacement size is associated with ModR/M and SIB byte, can't easily
	// encode the size information in operand type. So store it here.
	DispSize byte
}

func (insn *Instruction) set1Operand(op int, src byte) {
	insn.Opcode = op
	insn.Src = src
	insn.Noperand = 1
}

func (insn *Instruction) set2Operand(op int, src, dst byte) {
	insn.Opcode = op
	insn.Src = src
	insn.Dst = dst
	insn.Noperand = 2
}

const regToRm = 0

func (insn *Instruction) set2OperandModRM(op int, wField, dField byte) {
	if dField == regToRm {
		insn.set2Operand(op, OperandRegByte-wField, OperandRm)
	} else {
		insn.set2Operand(op, OperandRm, OperandRegByte-wField)
	}
}

func (insn *Instruction) insnOperandSize() byte {
	return insn.sizeOverride & 0x0f
}

func (insn *Instruction) insnAddressSize() byte {
	return insn.sizeOverride & 0x0f
}

func (insn *Instruction) setInsnOperandSize(v byte) {
	insn.sizeOverride |= v
}

func (insn *Instruction) setInsnAddressSize(v byte) {
	insn.sizeOverride |= v << 4
}

// Disassemble. Record information in each pass.
type DisContext struct {
	// Record position in the binary code
	binary io.ReaderAt
	offset int64

	Dflag     bool // Affects the operand-size and address-size attributes
	Protected bool // in Protected mode?

	OperandSize byte // These should be set when Dflag and Protected bit is
	AddressSize byte // changed

	Instruction
}

// Create a new DisContext with protected mode on, dflag set.
func NewDisContext(binary io.ReaderAt) (dc *DisContext) {
	dc = new(DisContext)

	dc.binary = binary
	dc.offset = 0
	dc.Dflag = true
	dc.Protected = true
	dc.OperandSize = OpSizeLong
	dc.AddressSize = OpSizeLong

	return
}

// Convert byte to int. true = 1, false = 0
func Btoi(b bool) int {
	if b {
		return 1
	}
	return 0
}

func (dc *DisContext) updateOperandAddressSize() {
	size := byte(OpSizeWord)
	if dc.Protected {
		size += byte(Btoi(dc.Dflag))
	}
	dc.OperandSize = size
	dc.AddressSize = size
}

func (dc *DisContext) EffectiveOperandSize() (size byte) {
	size = dc.OperandSize
	if dc.sizeOverride != 0 && dc.insnOperandSize() != 0 {
		size = dc.insnOperandSize()
	}
	return
}

func (dc *DisContext) EffectiveAddressSize() (size byte) {
	size = dc.AddressSize
	if dc.sizeOverride != 0 && dc.insnAddressSize() != 0 {
		size = dc.insnAddressSize()
	}
	return
}

func (dc *DisContext) SetDflag(v bool) {
	if v == dc.Dflag {
		return
	}
	dc.Dflag = v
	dc.updateOperandAddressSize()
}

func (dc *DisContext) SetProtected(v bool) {
	if v == dc.Protected {
		return
	}
	dc.Protected = v
	dc.updateOperandAddressSize()
}

// Parse 1 instruction. Return nil if no more data available.
func (dc *DisContext) NextInsn() *DisContext {
	// Error handling
	defer func() {
		if err := recover(); err != nil {
			if err != io.EOF {
				log.Println("work failed:", err)
			}
			return
		}
	}()
	dc.DispSize = 0
	dc.Scale = 0
	dc.Prefix = 0
	dc.sizeOverride = 0

	dc.parsePrefix()
	dc.parseOpcode()
	return dc
}

/* Reading binary */

var readBuf = [...][]byte{
	nil,
	make([]byte, 1),
	make([]byte, 2),
	make([]byte, 4),
}

// Size can only be OpSizeByte/Word/Long
func (dc *DisContext) readNBytes(size byte) (val int32) {
	n, err := dc.binary.ReadAt(readBuf[size], dc.offset)
	if err != nil {
		panic(err)
	}

	switch size {
	case OpSizeByte:
		val = int32(readBuf[size][0])
	case OpSizeWord:
		val = int32(binary.LittleEndian.Uint16(readBuf[size]))
	case OpSizeLong:
		val = int32(binary.LittleEndian.Uint32(readBuf[size]))
	}

	dc.offset += int64(n)
	return
}

// Get the next byte in the instruction stream
func (dc *DisContext) nextByte() byte {
	return byte(dc.readNBytes(OpSizeByte))
}

func (dc *DisContext) nextWord() int16 {
	return int16(dc.readNBytes(OpSizeWord))
}

func (dc *DisContext) nextLong() int32 {
	return dc.readNBytes(OpSizeLong)
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
	// XXX Is the addressing mode of ModR/M byte affected by the address-size
	// attribute?
	switch dc.AddressSize {
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
	if dc.Mod == 3 {
		return
	}

	if dc.Rm == 4 {
		dc.parseSIB()
	}

	switch dc.Mod {
	case 0:
		if dc.Rm == 5 {
			dc.Disp = int32(dc.nextLong())
			dc.DispSize = OpSizeLong
		}
	case 1:
		dc.Disp = int32(dc.nextByte())
		dc.DispSize = OpSizeByte
	case 2:
		dc.Disp = int32(dc.nextLong())
		dc.DispSize = OpSizeLong
	}
}

func (dc *DisContext) parseAfterModRM16bit() {
	// Refer to Intel Manual 2A Table 2-1
	switch dc.Mod {
	case 0:
		if dc.Rm == 6 {
			dc.getDisp(OpSizeWord)
		}
	case 1:
		dc.getDisp(OpSizeByte)
	case 2:
		dc.getDisp(OpSizeWord)
	}
}

func (dc *DisContext) parseSIB() {
	// SIB has the same bit field allocation with ModR/M byte
	dc.Scale, dc.Index, dc.Base = parseBitField(dc.nextByte())
	dc.Scale = 1 << dc.Scale

	if dc.Base == 5 {
		switch dc.Mod {
		case 0, 2:
			dc.getDisp(OpSizeLong)
		case 1:
			dc.getDisp(OpSizeByte)
		}
	}
}

/* Immediate value */

func (dc *DisContext) getImmediate(ot byte) {
	switch ot {
	case OperandImm:
		dc.ImmOff = dc.readNBytes(dc.EffectiveOperandSize())
	case OperandImmByte:
		dc.ImmOff = int32(dc.nextByte())
	default:
		log.Fatalf("Immediate operand type wrong %d\n", ot)
	}
}

func (dc *DisContext) getDisp(size byte) {
	dc.DispSize = size
	dc.Disp = int32(dc.readNBytes(size))
}

// Get memory offset
func (dc *DisContext) getMOffset() {
	dc.ImmOff = int32(dc.readNBytes(dc.EffectiveAddressSize()))
}
