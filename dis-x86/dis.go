package dis

import (
	"encoding/binary"
	"fmt"
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
	OpSizeNone byte = iota // zero means not defined
	OpSizeByte
	OpSizeWord
	OpSizeLong // Long = DoubleWord
	OpSizeQuad
	OpSizeFull // means size depend on operand-size
)

type InsnInfo struct {
	OpId byte
	Flag uint64 // Contains information about how to parse the instruction
	// The operand type is defined in insn.go. Look at diStorm's instructions.h
	// for the meaning of each operand type.
	Operand [4]byte
}

type Instruction struct {
	Prefix int
	Info   *InsnInfo

	Disp   int32 // Displacement. For lgdt and related, this is the limit
	ImmOff int32 // Immediate value or Offset. For lgdt and related, this is base

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
	binary    io.ReaderAt
	offset    int64 // Record position in the binary code
	insnStart int64 // Begin offset of the current instruction

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
	dc.insnStart = dc.offset

	dc.parsePrefix()
	dc.parseOpcode()
	return dc
}

func (dc *DisContext) parseOpcode() {
	opcode := dc.nextByte()
	opcodeAll := int(opcode)

	// If this is a escape, we need to access InsnDB2 using the second opcode byte
	if opcode != 0x0f {
		dc.Info = &InsnDB[opcode]
		// debug.Printf("opcode: %#02x\n", opcode)
	} else {
		opcode = dc.nextByte()
		dc.Info = &InsnDB2[opcode]
		opcodeAll = opcodeAll<<8 + int(opcode)
		// debug.Printf("opcode: 0x0f, %#02x\n", opcodeAll)
	}
	if dc.Info.OpId == 0 {
		panic(fmt.Sprintf("No such opcode %#x", opcodeAll))
	}

	if dc.Info.Flag&IFLAG_MODRM_REQUIRED != 0 {
		// debug.Println("parse modrm")
		dc.parseModRM()
	}
	if dc.Info.Flag&IFLAG_MODRM_INCLUDED != 0 {
		// Because of Go's address operator's limitation, we first find the
		// index in the grpInsnInfoIndex, then use the index to access the
		// grpInsnInfo array.
		opcodeAll = opcodeAll<<8 + int(dc.Reg)
		idx, ok := grpInsnInfoIndex[opcodeAll]
		if !ok {
			panic(fmt.Sprintf("Group instruction key %#x lookup failed", opcodeAll))
		}
		dc.Info = &(grpInsnInfo[idx])
		// debug.Printf("Opcode: %#02x reg field %#x used as insn encoding, OpId: %#02x", opcodeAll, dc.Reg, dc.Info.OpId)
	}
	dc.parseOperand(opcode)
}

func (dc *DisContext) parseOperand(opcode byte) {
	for _, op := range dc.Info.Operand {
		if op == OT_NONE {
			break
		}

		switch byte(op) {
		case OT_REGI_EDI:
			dc.Reg = Edi
		case OT_ACC8, OT_ACC16, OT_ACC_FULL:
			// debug.Println("parseOperand eax as reg")
			dc.Reg = Eax
		// Immediate value
		case OT_IMM8, OT_IMM16, OT_IMM32:
			// debug.Println("parseOperand read immediate")
			dc.ImmOff = dc.readNBytes(ot2size[op])
		case OT_IMM_FULL:
			// debug.Println("parseOperand read full immediate")
			dc.ImmOff = dc.readNBytes(dc.EffectiveOperandSize())

		// Instruction block (opcode) contains reg field
		case OT_IB_R_FULL, OT_IB_RB:
			// debug.Println("parseOperand instruction block contains reg field")
			dc.Reg = opcode & 0x7
		case OT_SEG:
			// debug.Println("parseOperand opcode lowest 3 bits contains reg field")
			dc.Reg = opcode >> 3 & 0x03

		// Memory offset. Only used by mov (memory offset)
		case OT_MOFFS8, OT_MOFFS_FULL:
			// According to Intel Manual, the size of the offset is affected
			// by address-size attribute. The size of the data is either
			// determinied by the instruction itself or operand-size
			// attribute.
			// debug.Println("parseOperand moffset")
			dc.ImmOff = dc.readNBytes(dc.EffectiveAddressSize())

		// Relative code offset
		case OT_RELCB:
			dc.ImmOff = int32(dc.nextByte())
			// debug.Printf("RECB: %#x\n", dc.ImmOff)

		// sign-extended 8-bit immediate
		case OT_SEIMM8:
			dc.ImmOff = int32(dc.nextByte())
		}
	}
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
			dc.getDisp(OpSizeLong)
		}
	case 1:
		dc.getDisp(OpSizeByte)
	case 2:
		dc.getDisp(OpSizeLong)
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

	// debug.Println("parseModRM")
	if dc.Base == 5 {
		switch dc.Mod {
		case 0, 2:
			dc.getDisp(OpSizeLong)
		case 1:
			dc.getDisp(OpSizeByte)
		}
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

func init() {
	// Additional set up for operand type to size mapping
	ot2size[OT_IB_RB] = OpSizeByte
	ot2size[OT_REGI_EDI] = OpSizeFull
}
