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
	OpDec:  "dec",
}

var regName = [...]string{
	Eax: "ax",
	Ecx: "cx",
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
func formatReg(reg, size byte) (name string) {
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

func formatMemReg(reg, size byte) (name string) {
	return fmt.Sprintf("(%s)", formatReg(reg, size))
}

func (dc *DisContext) dumpDisp() string {
	return fmt.Sprintf("%#x", dc.Disp)
}

func (dc *DisContext) dumpImm() string {
	return fmt.Sprintf("$%#x", dc.Imm)
}

func (dc *DisContext) dumpReg() string {
	return formatReg(dc.Reg, dc.OperandSize)
}

func (dc *DisContext) dumpRm() (dump string) {
	if dc.AddressSize == OpSizeLong {
		return dc.dumpRm32bit()
	}
	return "not supported"
}

func (dc *DisContext) dumpRm32bit() (dump string) {
	if dc.Mod == 3 {
		return formatReg(dc.Rm, dc.OperandSize)
	}

	if dc.hasDisp {
		dump = dc.dumpDisp()
	}
	if dc.hasSIB {
		dump += dc.dumpSIB()
	} else if !(dc.Rm == 5 && dc.Mod == 0) {
		dump += formatMemReg(dc.Rm, dc.OperandSize)
	}
	return
}

func (dc *DisContext) dumpSIB() string {
	// Refer to Intel Manual 2A Table 2-3
	var scale, base, index string

	if !(dc.Base == 5 && dc.Mod == 0) {
		base = formatReg(dc.Base, dc.OperandSize)
	}

	if dc.Index != 4 {
		// XXX What does none mean for scale index? Only use the base register
		// in SIB?
		index = formatReg(dc.Index, dc.OperandSize)
		scale = fmt.Sprintf("%d", 1<<dc.Scale)
	}
	return fmt.Sprintf("(%s,%s,%s)", base, index, scale)
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
	var src, dst string
	op := dc.RawOpCode[0]
	if op&0x0f < 2 {
		src = dc.dumpReg()
		dst = dc.dumpRm()
	} else {
		src = dc.dumpRm()
		dst = dc.dumpReg()
	}

	return fmt.Sprintf("%s %s,%s", insnName[dc.Opcode], src, dst)
}

func dumpInsnImmReg(dc *DisContext) string {
	return fmt.Sprintf("%s %s,%s", insnName[dc.Opcode],
		dc.dumpImm(), dc.dumpReg())
}

func dumpInsnReg(dc *DisContext) string {
	return fmt.Sprintf("%s %s", insnName[dc.Opcode], dc.dumpReg())
}
