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
	/* Arithmetic and logic instructions */

	// arith
	case 0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d:
		fallthrough
	case 0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x18, 0x19, 0x1a, 0x1b, 0x1c, 0x1d:
		fallthrough
	case 0x20, 0x21, 0x22, 0x23, 0x24, 0x25, 0x28, 0x29, 0x2a, 0x2b, 0x2c, 0x2d:
		fallthrough
	case 0x30, 0x31, 0x32, 0x33, 0x34, 0x35, 0x38, 0x39, 0x3a, 0x3b, 0x3c, 0x3d:
		dc.parseArith(op)

	// inc
	case 0x40, 0x41, 0x42, 0x43, 0x44, 0x45, 0x46, 0x47:
		dc.Reg = op - 0x40
		dc.set1Operand(OpInc, OperandReg)
	// dec
	case 0x48, 0x49, 0x4a, 0x4b, 0x4c, 0x4d, 0x4e, 0x4f:
		dc.Reg = op - 0x48
		dc.set1Operand(OpDec, OperandReg)

	/* Stack instructions */

	// segment related push/pop
	case 0x06, 0x16, 0x07, 0x17, 0x0e, 0x1e, 0x1f:
		segopmap := segStackOpcode[op]
		dc.Reg = segopmap[1]
		dc.set1Operand(int(segopmap[0]), OperandReg)

	// push
	case 0x50, 0x51, 0x52, 0x53, 0x54, 0x55, 0x56, 0x57:
		dc.Reg = op - 0x50
		dc.set1Operand(OpPush, OperandReg)
	// pop
	case 0x58, 0x59, 0x5a, 0x5b, 0x5c, 0x5d, 0x5e, 0x5f:
		dc.Reg = op - 0x58
		dc.set1Operand(OpPop, OperandReg)

	/* Memory instructions */

	// mov offset,ax (0xa0, a1)
	// mov ax,offset (0xa2, a3)
	case 0xa0, 0xa1, 0xa2, 0xa3:
		dc.parseMovEax(op)

	// mov (immediate byte into byte register)
	case 0xb0, 0xb1, 0xb2, 0xb3, 0xb4, 0xb5, 0xb6, 0xb7:
		dc.ImmOff = int32(dc.nextByte())
		dc.Reg = op - 0xb0
		dc.setInsnOperandSize(OpSizeByte)
		dc.set2Operand(OpMov, OperandImm, OperandReg)
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

	switch l {
	case 0, 1, 2, 3:
		dc.parseModRM()
		if l < 2 {
			dc.set2Operand(opcode, OperandReg, OperandRm)
		} else {
			dc.set2Operand(opcode, OperandRm, OperandReg)
		}
	case 4:
		dc.Reg = Al
		dc.ImmOff = int32(dc.nextByte())
		dc.set2Operand(opcode, OperandImm, OperandReg)
	case 5:
		dc.Reg = Eax
		dc.ImmOff = dc.getImmediate()
		dc.set2Operand(opcode, OperandImm, OperandReg)
	default:
		log.Panicln("parseArith: byte 0x%x: error", op)
	}
}

var segStackOpcode = map[byte]([2]byte){
	0x06: [2]byte{OpPush, ES}, 0x07: [2]byte{OpPop, ES},
	0x16: [2]byte{OpPush, SS}, 0x17: [2]byte{OpPop, SS},
	0x0e: [2]byte{OpPush, CS},       // 0x0f: 2 byte opcode escape
	0x1e: [2]byte{OpPush, DS}, 0x1f: [2]byte{OpPop, DS},
}

var movA0A3Table = map[byte]([2]byte){
	0xa0: [2]byte{OperandMOffByte, OperandReg},
	0xa1: [2]byte{OperandMOffCalc, OperandReg},
	0xa2: [2]byte{OperandReg, OperandMOffByte},
	0xa3: [2]byte{OperandReg, OperandMOffCalc},
}

func (dc *DisContext) parseMovEax(op byte) {
	te := movA0A3Table[op]
	dc.ImmOff = int32(dc.nextLong())
	dc.Reg = Eax
	dc.set2Operand(OpMov, te[0], te[1])
}
