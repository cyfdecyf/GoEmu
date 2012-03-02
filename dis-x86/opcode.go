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
		parseArith(op, dc)

	// inc
	case 0x40, 0x41, 0x42, 0x43, 0x44, 0x45, 0x46, 0x47:
		dc.Opcode = OpInc
		dc.Reg = op - 0x40
	// dec
	case 0x48, 0x49, 0x4a, 0x4b, 0x4c, 0x4d, 0x4e, 0x4f:
		dc.Opcode = OpDec
		dc.Reg = op - 0x48

	/* Stack instructions */

	// segment related push/pop
	case 0x06, 0x16, 0x07, 0x17, 0x0e, 0x1e, 0x1f:
		segopmap := segStackOpcode[op]
		dc.Opcode, dc.Reg = int(segopmap[0]), segopmap[1]

	// push
	case 0x50, 0x51, 0x52, 0x53, 0x54, 0x55, 0x56, 0x57:
		dc.Opcode = OpPush
		dc.Reg = op - 0x50
	// pop
	case 0x58, 0x59, 0x5a, 0x5b, 0x5c, 0x5d, 0x5e, 0x5f:
		dc.Opcode = OpPop
		dc.Reg = op - 0x58
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

func parseArith(op byte, dc *DisContext) {
	h, l := op>>4, op&0x0f
	if l < 8 {
		dc.Opcode = arithOpcode1[h]
	} else {
		dc.Opcode = arithOpcode2[h]
	}

	switch l {
	case 0, 1, 2, 3:
		dc.parseModRM()
	case 4:
		dc.Reg = Al
		dc.Imm = int32(dc.nextByte())
	case 5:
		dc.Reg = Eax
		dc.Imm = dc.getImmediate()
	default:
		log.Panicln("parseArith: byte 0x%x: error", op)
	}
}

var segStackOpcode = map[byte]([2]byte){
	0x06: [2]byte{OpPush, ES}, 0x07: [2]byte{OpPop, ES},
	0x16: [2]byte{OpPush, SS}, 0x17: [2]byte{OpPop, SS},
	0x0e: [2]byte{OpPush, CS},
	0x1e: [2]byte{OpPush, DS}, 0x1f: [2]byte{OpPop, DS},
}
