package dis

import (
	"bytes"
	"fmt"
	"log"
)

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

var segRegName = [...]string{
	ES: "es",
	CS: "cs",
	SS: "ss",
	DS: "ds",
	FS: "fs",
	GS: "gs",
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
func (dc *DisContext) formatReg(reg byte, size byte) (name string) {
	if size == OpSizeFull {
		size = dc.EffectiveOperandSize()
	}
	// debug.Println("size:", size, "reg:", reg)
	switch size {
	case OpSizeByte:
		name = regName8[reg]
	case OpSizeWord:
		name = regName[reg]
	case OpSizeLong:
		name = "e" + regName[reg]
		// debug.Println(name)
	case OpSizeQuad:
		name = "r" + regName[reg]
	default:
		log.Fatalf("reg size %d not correct\n", size)
	}
	return "%" + name
}

func (dc *DisContext) dumpReg(size byte) string {
	if size == OpSizeFull {
		size = dc.EffectiveOperandSize()
	}
	return dc.formatReg(dc.Reg, size)
}

func dumpSignedValue(size byte, val int32) (dump string) {
	switch size {
	case OpSizeByte:
		dump = fmt.Sprintf("%#x", int8(val))
	case OpSizeWord:
		dump = fmt.Sprintf("%#x", int16(val))
	case OpSizeLong:
		dump = fmt.Sprintf("%#x", val)
	}
	return
}

func (dc *DisContext) dumpDisp() (dump string) {
	return dumpSignedValue(dc.DispSize, dc.Disp)
}

func (dc *DisContext) dumpImm() (dump string) {
	return fmt.Sprintf("$%#x", uint32(dc.ImmOff))
}

func (dc *DisContext) dumpRm(operandSize, addressSize byte) (dump string) {
	if dc.Mod == 3 {
		// debug.Println("modrm = 3")
		if operandSize == OpSizeFull {
			operandSize = dc.EffectiveOperandSize()
		}
		// debug.Println("operandSize:", operandSize)
		return dc.formatReg(dc.Rm, operandSize)
	}

	if addressSize == OpSizeFull {
		addressSize = dc.EffectiveAddressSize()
	}
	switch addressSize {
	case OpSizeLong:
		dump = dc.dumpRm32bit()
	case OpSizeWord:
		dump = dc.dumpRm16bit()
	}
	return
}

func (dc *DisContext) dumpRm32bit() (dump string) {
	// First output displacement
	if dc.DispSize != 0 {
		dump = dc.dumpDisp()
	}
	if dc.Scale != 0 {
		dump += dc.dumpSIB()
	} else if !(dc.Rm == 5 && dc.Mod == 0) {
		dump += fmt.Sprintf("(%s)", dc.formatReg(dc.Rm, OpSizeLong))
	}
	return
}

func (dc *DisContext) dumpRm16bit() (dump string) {
	if dc.DispSize != 0 {
		dump = dc.dumpDisp()
	}
	panic("16bit modrm not supported now.")
	return
}

func (dc *DisContext) dumpSIB() string {
	// Refer to Intel Manual 2A Table 2-3
	var scale, base, index string

	if !(dc.Base == 5 && dc.Mod == 0) {
		// SIB is only allowed in 32-bit mode
		base = dc.formatReg(dc.Base, OpSizeLong)
	}

	if dc.Index != 4 {
		// XXX What does none mean for scale index? Only use the base register
		// in SIB?
		index = dc.formatReg(dc.Index, OpSizeLong)
		scale = fmt.Sprintf("%d", dc.Scale)
	} else if dc.Info.OpId == Insn_Lea {
		// Don't know why objdump uses "%eiz" when there's no index and scale
		index = "%eiz"
		scale = "1"
	}
	if index != "" || scale != "" {
		return fmt.Sprintf("(%s,%s,%s)", base, index, scale)
	}
	return fmt.Sprintf("(%s)", base)
}

var insnSizeSuffix = []string{
	OT_RM8:     "b",
	OT_RM_FULL: "l",
}

func (dc *DisContext) dumpInsn() (dump string) {
	dump = InsnName[dc.Info.OpId]
	switch dc.Info.OpId {
	case Insn_Test:
		// For test (0xf6), operand size is always 8bit. But when dumping
		// ModRM with memory reference, we always use 32bit register. I guess
		// this is why objdump appends the 'b' suffix, it makes us easier to
		// know the operand size from the instruction name.
		if dc.Mod != 3 {
			dump += insnSizeSuffix[dc.Info.Operand[0]]
		}
	case Insn_Lgdt, Insn_Sgdt:
		dump += "l"
	case Insn_Cmp:
		switch dc.Info.Operand[0] {
		case OT_RM8:
			dump += "b"
		}
	case Insn_Mov:
		if dc.Mod != 3 {
			if dc.opcode == 0xc7 || dc.opcode == 0xc6 {
				dump += insnSizeSuffix[dc.Info.Operand[0]]
			}
		}
	}
	return dump + " "
}

func (dc *DisContext) dumpPrefix() string {
	if dc.Prefix&(PrefixREPZ|PrefixREPNZ) != 0 {
		return "rep "
	}
	return ""
}

func (dc *DisContext) DumpInsn() (dump string) {
	var buf bytes.Buffer

	buf.WriteString(dc.dumpPrefix())

	if dumper, ok := specialInsnDump[dc.Info.OpId]; ok == true {
		buf.WriteString(dumper(dc))
		return buf.String()
	}

	buf.WriteString(dc.dumpInsn())
	switch dc.Info.countOperand() {
	case 1:
		buf.WriteString(dc.dumpOperand(dc.Info.Operand[0]))
	case 2:
		buf.WriteString(dc.dumpOperand(dc.Info.Operand[1]))
		buf.WriteString(",")
		buf.WriteString(dc.dumpOperand(dc.Info.Operand[0]))
	}
	return buf.String()
}

func (dc *DisContext) dumpOperand(operand byte) (dump string) {
	switch operand {
	// Immediate value
	case OT_IMM8, OT_IMM16, OT_IMM32, OT_IMM_FULL, OT_SEIMM8:
		dump = dc.dumpImm()

	// Memory offset are always unsigned
	case OT_MOFFS8, OT_MOFFS_FULL:
		dump = fmt.Sprintf("%#x", uint32(dc.ImmOff))

	// Register
	case OT_REG8, OT_IB_RB, OT_ACC8,
		OT_REG16, OT_ACC16,
		OT_REG32,
		OT_REG_FULL, OT_IB_R_FULL, OT_ACC_FULL,
		OT_REGI_EDI:
		// debug.Println("dump reg")
		dump = dc.dumpReg(ot2size[operand])
	// Segment register
	case OT_SREG, OT_SEG:
		dump = "%" + segRegName[dc.Reg]

	// RM
	// RM8 means the operand size is 8, but is the same with RM_FULL for
	// address, which depends on address-size attribute.
	// Example: mov (0x88) -- RM8, mov (0x89) -- RM_FULL
	case OT_RM8, OT_RM_FULL, OT_MEM:
		// debug.Println("dump rm")
		dump = dc.dumpRm(ot2size[operand], OpSizeFull)
	// Some instruction forces 16 bit addressing. Exmaple: arpl (0x63)
	case OT_RM16:
		dump = dc.dumpRm(ot2size[operand], OpSizeWord)
	// Messy x86, sigh. If the operand is register, use 32bit; if it's memory, use 16 bit.
	// Example: mov (0x8c), when used as register, 32bit, but for memory, 16 bit memory
	case OT_RFULL_M16:
		dump = dc.dumpRm(OpSizeLong, OpSizeWord)

	case OT_MEM16_3264:
		// What operand size should we use here?
		dump = dc.dumpRm(OpSizeLong, OpSizeLong)
	}

	return
}

// Some intructions are difficult to dump because the format returned by
// objdump is not regular. For those instructions, I just use specific dump
// function for each instruction.

type insnDumper func(dc *DisContext) string

var specialInsnDump = map[byte]insnDumper{
	Insn_Stos: dumpStos,
	Insn_Movs: dumpMovs,
}

func dumpStos(dc *DisContext) (dump string) {
	switch dc.EffectiveAddressSize() {
	case OpSizeWord:
		panic("not implemented")
	case OpSizeLong:
		dump = "stos %eax,%es:(%edi)"
	}
	return
}

func dumpMovs(dc *DisContext) (dump string) {
	switch dc.EffectiveAddressSize() {
	case OpSizeWord:
		panic("not implemented")
	case OpSizeLong:
		dump = "movsl %ds:(%esi),%es:(%edi)"
	}
	return
}
