package dis

import (
	"fmt"
	"log"
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
	Al: "al",
	Cl: "cl",
	Dl: "dl",
	Bl: "bl",
	Ah: "ah",
	Ch: "ch",
	Dh: "dh",
	Bh: "bh",
}

type insnDumper func(string, *Instrucion) string

// Return the string name of a register
func formatRegister(reg, size byte) (name string) {
	switch size {
	case OpSizeByte:
		name = regName8[reg]
	case OpSizeWord:
		name = regName[reg]
	case OpSizeLong:
		name = "%e" + regName[reg]
	case OpSizeQuad:
		name = "%r" + regName[reg]
	default:
		log.Fatalf("operand size %d not correct\n", size)
	}
	return
}

func (dc *DisContext) DumpInsn() (dump string) {
	switch dc.RawOpCode[0] {
	// arith
	case 0x00, 0x01, 0x02, 0x03, 0x08, 0x09, 0x0a, 0x0b:
		fallthrough
	case 0x10, 0x11, 0x12, 0x13, 0x18, 0x19, 0x1a, 0x1b:
		fallthrough
	case 0x20, 0x21, 0x22, 0x23, 0x28, 0x29, 0x2a, 0x2b:
		fallthrough
	case 0x30, 0x31, 0x32, 0x33, 0x38, 0x39, 0x3a, 0x3b:
		dump = dumpInsnModRM(dc)

	// arith
	case 0x04, 0x05, 0x0c, 0x0d:
		fallthrough
	case 0x14, 0x15, 0x1c, 0x1d:
		fallthrough
	case 0x24, 0x25, 0x2c, 0x2d:
		fallthrough
	case 0x34, 0x35, 0x3c, 0x3d:
		dump = dumpInsnImmReg(dc)

	// inc, dec
	case 0x40, 0x41, 0x42, 0x43, 0x44, 0x45, 0x46, 0x47:
		fallthrough
	case 0x48, 0x49, 0x4a, 0x4b, 0x4c, 0x4d, 0x4e, 0x4f:
		dump = dumpInsnReg(dc)
	}
	return
}

func dumpInsnModRM(dc *DisContext) string {
	var src, dst byte
	op := dc.RawOpCode[0]
	if op&0x0f < 2 {
		src, dst = dc.Rm, dc.Reg
	} else {
		src, dst = dc.Reg, dc.Rm
	}

	size := dc.calcOperandSize(CalcOpSize)
	return fmt.Sprintf("%s %s,%s", insnName[dc.Opcode],
		formatRegister(src, size), formatRegister(dst, size))
}

func dumpInsnImmReg(dc *DisContext) string {
	size := dc.calcOperandSize(CalcOpSize)
	return fmt.Sprintf("%s %d,%s", insnName[dc.Opcode],
		dc.Imm, formatRegister(dc.Reg, size))
}

func dumpInsnReg(dc *DisContext) string {
	size := dc.calcOperandSize(CalcOpSize)
	return fmt.Sprintf("%s %s", insnName[dc.Opcode],
		formatRegister(dc.Reg, size))
}
