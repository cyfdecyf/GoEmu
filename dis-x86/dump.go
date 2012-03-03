package dis

import (
	"fmt"
	"log"
)

var insnName = [...]string{
	OpAdd:  "add",
	OpMov:  "mov",
	OpPush: "push",
	OpPop:  "pop",
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
	ES:  "es",
	SS:  "ss",
	CS:  "cs",
	DS:  "ds",
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

// Return the string name of a register
func (dc *DisContext) formatReg(reg byte) (name string) {
	if reg >= ES {
		return "%" + regName[reg]
	}
	switch dc.EffectiveOperandSize() {
	case OpSizeByte:
		name = regName8[reg]
	case OpSizeWord:
		name = regName[reg]
	case OpSizeLong:
		name = "e" + regName[reg]
	case OpSizeQuad:
		name = "r" + regName[reg]
	default:
		log.Fatalf("operand size %d not correct\n", dc.EffectiveOperandSize())
	}
	return "%" + name
}

func (dc *DisContext) dumpReg() string {
	return dc.formatReg(dc.Reg)
}

func (dc *DisContext) dumpDisp() string {
	return fmt.Sprintf("%#x", dc.Disp)
}

func (dc *DisContext) dumpImm() string {
	return fmt.Sprintf("$%#x", dc.ImmOff)
}

func (dc *DisContext) dumpOffset() string {
	// Offset are unsigned
	return fmt.Sprintf("%#x", uint32(dc.ImmOff))
}

func (dc *DisContext) dumpRm() (dump string) {
	if dc.EffectiveAddressSize() == OpSizeLong {
		return dc.dumpRm32bit()
	}
	return "not supported"
}

func (dc *DisContext) dumpRm32bit() (dump string) {
	if dc.Mod == 3 {
		return dc.formatReg(dc.Rm)
	}

	if dc.hasDisp {
		dump = dc.dumpDisp()
	}
	if dc.hasSIB {
		dump += dc.dumpSIB()
	} else if !(dc.Rm == 5 && dc.Mod == 0) {
		dump += fmt.Sprintf("(%s)", dc.formatReg(dc.Rm))
	}
	return
}

func (dc *DisContext) dumpSIB() string {
	// Refer to Intel Manual 2A Table 2-3
	var scale, base, index string

	if !(dc.Base == 5 && dc.Mod == 0) {
		base = dc.formatReg(dc.Base)
	}

	if dc.Index != 4 {
		// XXX What does none mean for scale index? Only use the base register
		// in SIB?
		index = dc.formatReg(dc.Index)
		scale = fmt.Sprintf("%d", 1<<dc.Scale)
	}
	return fmt.Sprintf("(%s,%s,%s)", base, index, scale)
}

func (dc *DisContext) DumpInsn() (dump string) {
	switch dc.Noperand {
	case 0:
		dump = dc.dump0OpInsn()
	case 1:
		dump = dc.dump1OpInsn()
	case 2:
		dump = dc.dump2OpInsn()
	default:
		log.Fatalln("Operand size not correct or supported.")
	}
	return
}

func (dc *DisContext) dumpOperand(operand byte) (dump string) {
	switch operand {
	case OperandMOffByte, OperandMOffCalc:
		dump = dc.dumpOffset()
	case OperandImm:
		dump = dc.dumpImm()
	case OperandReg:
		dump = dc.dumpReg()
	case OperandRm:
		dump = dc.dumpRm()
	}
	return
}

func (dc *DisContext) dump0OpInsn() string {
	return insnName[dc.Opcode]
}

func (dc *DisContext) dump1OpInsn() string {
	return fmt.Sprintf("%s %s", insnName[dc.Opcode], dc.dumpOperand(dc.Src))
}

func (dc *DisContext) dump2OpInsn() string {
	return fmt.Sprintf("%s %s,%s", insnName[dc.Opcode],
		dc.dumpOperand(dc.Src), dc.dumpOperand(dc.Dst))
}
