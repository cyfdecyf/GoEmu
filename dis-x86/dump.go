package dis

import (
	"fmt"
	"log"
)

const (
	opSizeByte = iota
	opSizeWord
	opSizeLong
	opSizeQuad
)

var insnName = [...]string{
	OpAdd:  "add",
	OpMov:  "mov",
	OpPush: "push",
	OpLea:  "lea",
	OpRet:  "ret",
}

var regName = [...]string{
	Eax: "ax",
	Ecx: "bx",
	Edx: "dx",
	Ebx: "bx",
	Esp: "sp",
	Ebp: "bp",
	Esi: "si",
	Edi: "di",
}

var regName8 = [...]string{
	Eax: "al",
	Ecx: "cl",
	Edx: "dl",
	Ebx: "bl",
}

type insnDumper func(*DisContext) string

var opcodeDumper = [...]insnDumper{
	// mov
	OpMov: dumpMov,
}

// Return the string name of a register
func formatRegister(reg, size int) (name string) {
	switch size {
	case opSizeByte:
		if reg > 3 {
			log.Panicf("8 bit reg index %d out of range\n", reg)
		}
		name = regName8[reg]
	case opSizeWord:
		name = regName[reg]
	case opSizeLong:
		name = "%e" + regName[reg]
	case opSizeQuad:
		name = "%r" + regName[reg]
	default:
		log.Panicf("operand size %d not correct\n", size)
	}
	return
}

func (dc *DisContext) DumpInsn() string {
	dumper := opcodeDumper[dc.Opcode]
	if dumper == nil {
		return insnName[dc.Opcode]
	}
	return dumper(dc)
}

func dumpMov(dc *DisContext) (dump string) {
	switch dc.Raw[0] {
	// Copies the second operand (source operand) to the first operand
	// (destination operand).
	case 0x88, 0x89, 0x8a, 0x8b:
		dump = fmt.Sprintf("%s %s,%s", insnName[dc.Opcode],
			formatRegister(dc.Operand2, opSizeLong),
			formatRegister(dc.Operand1, opSizeLong))
	}
	return
}
