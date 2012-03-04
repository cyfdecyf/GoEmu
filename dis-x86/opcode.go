package dis

import (
	"log"
)

const (
	// Arith
	OpAdd = iota
	OpAdc
	OpAnd
	OpXor
	OpOr
	OpSbb
	OpSub
	OpCmp

	OpInc // Keep the order of the 4 instructions
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
	case 0x00, 0x01, 0x02, 0x03, 0x04, 0x05, // add
		0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, // or
		0x10, 0x11, 0x12, 0x13, 0x14, 0x15, // adc
		0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d, // sbb
		0x20, 0x21, 0x22, 0x23, 0x24, 0x25, // and
		0x28, 0x29, 0x2a, 0x2b, 0x2c, 0x2d, // sub
		0x30, 0x31, 0x32, 0x33, 0x34, 0x35, // xor
		0x38, 0x39, 0x3a, 0x3b, 0x3c, 0x3d: // cmp
		dc.parseArith(op)

	// inc, dec, push, pop
	case 0x40, 0x41, 0x42, 0x43, 0x44, 0x45, 0x46, 0x47, // inc
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
		byteBit := op & 0x0f >> 3
		dc.getImmediate(OperandImmByte - byteBit)
		dc.set2Operand(OpMov, OperandImmByte-byteBit, OperandRegByte-byteBit)
	}
}

var arithOpcode1 = [...]int{
	0: OpAdd,
	1: OpAdc,
	2: OpAnd,
	3: OpXor,
}

var arithOpcode2 = [...]int{
	0: OpOr,
	1: OpSbb,
	2: OpSub,
	3: OpCmp,
}

func (dc *DisContext) parseArith(op byte) {
	h, l := op>>4, op&0x0f
	var opcode int
	if l < 8 {
		opcode = arithOpcode1[h]
	} else {
		opcode = arithOpcode2[h]
	}

	byteBit := l & 0x1
	switch l {
	case 0, 1, 2, 3:
		dc.parseModRM()
		if l&0x2 == 0 { // bit 2 indicates the direction
			dc.set2Operand(opcode, OperandRegByte-byteBit, OperandRm)
		} else {
			dc.set2Operand(opcode, OperandRm, OperandRegByte-byteBit)
		}
	case 4, 5:
		dc.Reg = Eax
		dc.getImmediate(OperandImmByte - byteBit)
		dc.set2Operand(opcode, OperandImmByte-byteBit, OperandRegByte-byteBit)
	default:
		log.Panicln("parseArith: byte 0x%x: error", op)
	}
}

var pushPopSegTable = map[byte]([2]byte){
	0x06: [2]byte{OpPush, ES}, 0x07: [2]byte{OpPop, ES},
	0x16: [2]byte{OpPush, SS}, 0x17: [2]byte{OpPop, SS},
	0x0e: [2]byte{OpPush, CS},       // 0x0f: 2 byte opcode escape
	0x1e: [2]byte{OpPush, DS}, 0x1f: [2]byte{OpPop, DS},
}

func (dc *DisContext) parsePushPopSeg(op byte) {
	te := pushPopSegTable[op]
	dc.Reg = te[1]
	dc.set1Operand(int(te[0]), OperandReg)
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

var movModRMTable = [...]([2]byte){
	// Starts from 0x88
	[2]byte{OperandRegByte, OperandRm},
	[2]byte{OperandReg, OperandRm},
	[2]byte{OperandRm, OperandRegByte},
	[2]byte{OperandRm, OperandReg},
}

func (dc *DisContext) parseMovModRM(op byte) {
	te := movModRMTable[op-0x88]
	dc.parseModRM()
	dc.set2Operand(OpMov, te[0], te[1])
}
