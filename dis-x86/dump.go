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
	// If the displacement is used alone, take it as unsigned value.
	if dc.Mod == 0 && dc.Rm == 5 {
		dump = fmt.Sprintf("%#x", uint32(dc.Disp))
	} else {
		dump = dumpSignedValue(dc.DispSize, dc.Disp)
	}
	return
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

	dump = dc.dumpSegPrefix()
	if addressSize == OpSizeFull {
		addressSize = dc.EffectiveAddressSize()
	}
	switch addressSize {
	case OpSizeLong:
		dump += dc.dumpRm32bit()
	case OpSizeWord:
		dump += dc.dumpRm16bit()
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
	OT_ACC8:       "",
	OT_RM8:        "b",
	OT_RM_FULL:    "l",
	OT_MEM16_3264: "l",
}

func (dc *DisContext) dumpInsn() (dump string) {
	dump = InsnName[dc.Info.OpId]

	// When the destination operand is memory address, and we can't infer
	// operand size directly from the src operand, add the appropriate suffix.
	// Example: test (0xf6), operand size is always 8bit. But when dumping
	// ModRM with memory reference, we always use 32bit register.
	if dc.Mod != 3 {
		switch dc.opcodeAll {
		case 0x8000, 0x8001, 0x8002, 0x8003, 0x8004, 0x8005, 0x8006, 0x8007, // Immediate Grp 1
			0x8100, 0x8101, 0x8102, 0x8103, 0x8104, 0x8105, 0x8106, 0x8107, // Immediate Grp 1
			0x8200, 0x8201, 0x8202, 0x8203, 0x8204, 0x8205, 0x8206, 0x8207, // Immediate Grp 1
			0x8300, 0x8301, 0x8302, 0x8303, 0x8304, 0x8305, 0x8306, 0x8307, // Immediate Grp 1
			0xf600, 0xf602, 0xf603, 0xf604, 0xf605, 0xf606, 0xf607, // Unary Grp 3
			0xf700, 0xf702, 0xf703, 0xf704, 0xf705, 0xf706, 0xf707, // Unary Grp 3
			0x0f0100, 0x0f0102, // sgdt, lgdt
			0xc600, 0xc700: // Grp 11 (mov)
			dump += insnSizeSuffix[dc.Info.Operand[0]]
		}
	}

	return dump + " "
}

var prefixName = map[int]string{
	PrefixREPNZ: "rep ",
	PrefixREPZ:  "rep ",
	PrefixCS:    "%cs:",
	PrefixSS:    "%ss",
	PrefixDS:    "%ds:",
	PrefixES:    "%es:",
	PrefixFS:    "%fs:",
	PrefixGS:    "%gs:",
	PrefixLOCK:  "lock ",
}

func (dc *DisContext) dumpRepLockPrefix() string {
	name, ok := prefixName[dc.Prefix&(PrefixREPNZ|PrefixREPZ|PrefixLOCK)]
	if ok {
		return name
	}
	return ""
}

func (dc *DisContext) dumpSegPrefix() string {
	name, ok := prefixName[dc.Prefix&(PrefixCS|PrefixDS|PrefixES|PrefixFS|PrefixGS)]
	if ok {
		return name
	}
	return ""
}

func (dc *DisContext) DumpInsn() (dump string) {
	var buf bytes.Buffer

	buf.WriteString(dc.dumpRepLockPrefix())

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
	case OT_REG8, OT_IB_RB, OT_REG16, OT_REG32,
		OT_REG_FULL, OT_IB_R_FULL,
		OT_ACC8, OT_ACC16, OT_ACC_FULL,
		OT_REGI_EDI, OT_REGCL:
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

	switch dc.opcodeAll {
	case 0xff02: // Call with indirect target
		dump = "*" + dump
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
