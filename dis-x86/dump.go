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
	OpInc:  "inc",
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

type insnDumper func(string, *Instrucion) string

var opcodeDumper = [...]insnDumper{
	OpMov: dumpMov,
	OpInc: dumpInc,
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

func DumpInsn(insn *Instrucion) string {
	dumper := opcodeDumper[insn.Opcode]
	name := insnName[insn.Opcode]
	if dumper == nil {
		return name
	}
	return dumper(name, insn)
}

func dumpMov(name string, insn *Instrucion) (dump string) {
	switch insn.Raw[0] {
	// Copies the second operand (source operand) to the first operand
	// (destination operand).
	case 0x88, 0x89, 0x8a, 0x8b:
		dump = fmt.Sprintf("%s %s,%s", name,
			formatRegister(insn.Operand2, opSizeLong),
			formatRegister(insn.Operand1, opSizeLong))
	}
	return
}

func dumpInc(name string, insn *Instrucion) (dump string) {
	return name + " " + formatRegister(insn.Operand1, opSizeLong)
}
