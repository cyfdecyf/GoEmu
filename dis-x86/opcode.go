package dis

import (
	"log"
)

const (
	// Keep the order of the instructions. Opcode parsing relies on the order.
	OpAdd = iota
	OpOr
	OpAdc
	OpSbb
	OpAnd
	OpSub
	OpXor
	OpCmp

	OpInc
	OpDec
	OpPush
	OpPop

	OpMov
	OpLea
	OpRet
)

type parseOp func(byte, *DisContext)

func (dc *DisContext) parseOpcode() {
	op := dc.nextByte()
	dc.RawOpCode[0] = op

	switch op {
	// Instruction will be grouped accord to function, and ordered
	// numerically.  But if it's possible to combine parsing, the group and
	// order rule will be broken.

	/* Arithmetic and logic instructions */

	// arith
	case
		0x00, 0x01, 0x02, 0x03, 0x04, 0x05, // add
		0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, // or
		0x10, 0x11, 0x12, 0x13, 0x14, 0x15, // adc
		0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, // sbb
		0x20, 0x21, 0x22, 0x23, 0x24, 0x25, // and
		0x28, 0x29, 0x2a, 0x2b, 0x2c, 0x2d, // sub
		0x30, 0x31, 0x32, 0x33, 0x34, 0x35, // xor
		0x38, 0x39, 0x3a, 0x3b, 0x3c, 0x3d: // cmp
		dc.parseArith(op)

	// inc, dec, push, pop
	case
		0x40, 0x41, 0x42, 0x43, 0x44, 0x45, 0x46, 0x47, // inc
		0x48, 0x49, 0x4a, 0x4b, 0x4c, 0x4d, 0x4e, 0x4f, // dec
		0x50, 0x51, 0x52, 0x53, 0x54, 0x55, 0x56, 0x57, // push
		0x58, 0x59, 0x5a, 0x5b, 0x5c, 0x5d, 0x5e, 0x5f: // pop
		dc.Reg = op & 0x07
		opId := int(op & 0x18 >> 3)
		dc.set1Operand(OpInc+opId, OperandReg)

	/* Stack instructions */

	// segment related push/pop
	case 0x06, 0x16, 0x07, 0x17, 0x0e, 0x1e, 0x1f:
		dc.parsePushPopSeg(op)

	/* Memory instructions */

	// mov reg
	case 0x88, 0x89, 0x8a, 0x8b:
		dc.parseMovModRM(op)
	// mov eax
	case 0xa0, 0xa1, 0xa2, 0xa3:
		dc.parseMovEax(op)
	case 0xb0, 0xb1, 0xb2, 0xb3, 0xb4, 0xb5, 0xb6, 0xb7, // mov (immediate byte into byte register)
		0xb8, 0xb9, 0xba, 0xbb, 0xbc, 0xdd, 0xbe, 0xbf: // mov (immediate word or long into byte register)
		dc.Reg = op & 0x07
		wField := op & 0x0f >> 3
		dc.getImmediate(OperandImmByte - wField)
		dc.set2Operand(OpMov, OperandImmByte-wField, OperandRegByte-wField)
	}
}

func (dc *DisContext) parseArith(op byte) {
	opcode := int(OpAdd + op&0x28)

	wField := op & 0x1
	switch op & 0xf {
	case 0, 1, 2, 3:
		dc.parseModRM()
		dc.set2OperandModRM(opcode, wField, op&0x02)
	case 4, 5:
		dc.Reg = Eax
		dc.getImmediate(OperandImmByte - wField)
		dc.set2Operand(opcode, OperandImmByte-wField, OperandRegByte-wField)
	default:
		log.Panicln("parseArith: byte 0x%x: error", op)
	}
}

func (dc *DisContext) parsePushPopSeg(op byte) {
	// Refer to Table B-13 on page B-18 of Vol 2C.
	opcode := OpPush + int(op&0x01)
	dc.Reg = op >> 3 & 0x3
	dc.set1Operand(opcode, OperandSegReg)
}

var movEaxTable = [...]([2]byte){
	// Starts from 0xa0
	[2]byte{OperandMOffByte, OperandRegByte},
	[2]byte{OperandMOff, OperandReg},
	[2]byte{OperandRegByte, OperandMOffByte},
	[2]byte{OperandReg, OperandMOff},
}

func (dc *DisContext) parseMovEax(op byte) {
	te := movEaxTable[op-0xa0]
	dc.getMOffset()
	dc.Reg = Eax
	dc.set2Operand(OpMov, te[0], te[1])
}

func (dc *DisContext) parseMovModRM(op byte) {
	dc.parseModRM()
	dc.set2OperandModRM(OpMov, op&0x01, op&0x02)
}
