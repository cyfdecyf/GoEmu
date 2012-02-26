package dis

import (
	"log"
)

const (
	OpAdd = iota
	OpInc
	OpPush
	OpPop
	OpMov
	OpLea
	OpRet
)

type parseOp func(byte, *DisContext)

var opcodeParser = [...]parseOp{
	// add
	0x00: parseAdd,
	0x01: parseAdd,
	0x02: parseAdd,
	0x03: parseAdd,
	0x04: parseAdd,
	0x05: parseAdd,
	0x80: parseAdd,
	0x81: parseAdd,
	0x83: parseAdd,

	// inc
	0x40: parseInc,
	0x41: parseInc,
	0x42: parseInc,
	0x43: parseInc,
	0x44: parseInc,
	0x45: parseInc,
	0x46: parseInc,
	0x47: parseInc,

	// push
	0x50: parsePush,
	0x51: parsePush,
	0x52: parsePush,
	0x53: parsePush,
	0x54: parsePush,
	0x55: parsePush,
	0x56: parsePush,
	0x57: parsePush,
	0xff: parsePush,
	0x6a: parsePush,
	0x68: parsePush,

	// pop
	0x58: parsePop,
	0x59: parsePop,
	0x5a: parsePop,
	0x5b: parsePop,
	0x5c: parsePop,
	0x5d: parsePop,
	0x5e: parsePop,
	0x5f: parsePop,

	// mov
	0x88: parseMov,
	0x89: parseMov,
	0x8a: parseMov,
	0x8b: parseMov,
	0x8c: parseMov,

	// lea
	0x8d: parseLea,

	// ret
	0xc3: parseRet,
	0xcb: parseRet,
	0xc2: parseRet,
	0xca: parseRet,
}

func (dc *DisContext) parseOpcode() {
	b := dc.getNextByte()
	parseFunc := opcodeParser[b]
	parseFunc(b, dc)
}

func parseInc(b byte, dc *DisContext) {
	dc.Opcode = OpInc
	dc.Operand1 = int(b) - 0x40
}

func parsePush(b byte, dc *DisContext) {
	dc.Opcode = OpPush

	switch b {
	case 0x50, 0x51, 0x52, 0x53, 0x54, 0x55, 0x56, 0x57:
		// Note the register order tricky makes this possible
		dc.Operand1 = int(b) - 0x50
	default:
		log.Panicln("parsePush: byte 0x%x: Not legal or not supported")
	}
}

func parsePop(b byte, dc *DisContext) {
	dc.Opcode = OpPop

	switch b {
	case 0x58, 0x59, 0x5a, 0x5b, 0x5c, 0x5d, 0x5e, 0x5f:
		dc.Operand1 = int(b) - 0x58
	}
}

func parseAdd(b byte, dc *DisContext) {
	dc.Opcode = OpAdd
	mod, reg, rm := parseModRM(dc.getNextByte())
	dc.Mod = mod

	switch b {
	case 0x00, 0x01, 0x02, 0x03:
		dc.Operand1 = reg
		dc.Operand2 = rm
	case 0x80, 0x81, 0x83:
		dc.Operand1 = rm
		// TODO Operand2 = imm8
	case 0x04, 0x05:
		dc.Operand1 = Eax
		// TODO Operand2 = imm8
	default:
		log.Panicln("parseAdd: byte 0x%x: Note legal or not supported", b)
	}
}

func parseLea(b byte, dc *DisContext) {
	dc.Opcode = OpLea
	mod, reg, rm := parseModRM(dc.getNextByte())
	dc.Mod = mod
	dc.Operand1 = reg
	dc.Operand2 = rm
}

func parseRet(b byte, dc *DisContext) {
	dc.Opcode = OpRet

	switch b {
	case 0xc3, 0xcb:
		// Do nothing
	case 0xc2, 0xca:
		// TODO Operand1 = imm16
	}
}

func parseMov(b byte, dc *DisContext) {
	dc.Opcode = OpMov
	mod, reg, rm := parseModRM(dc.getNextByte())
	dc.Mod = mod

	switch b {
	// Move register
	case 0x88, 0x89:
		dc.Operand1 = rm
		dc.Operand2 = reg
	case 0x8a, 0x8b:
		dc.Operand1 = reg
		dc.Operand2 = rm
	}
}
