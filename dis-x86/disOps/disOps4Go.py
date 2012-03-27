#!/usr/bin/env python

import x86sets
from x86header import *
import sys

"""
Generate instruction definition for GoEmu.

Much code is copied from x86db.py
"""

class DBException(Exception):
	""" Used in order to throw an exception when an error occurrs in the DB. """
	pass

class InstructionDB():
	def __init__(self):
		# Holds all the instructions, { "name" : opcodeId }
		self.name_opid = {}
		# Current largest instruction id. Opcode id is only used in Go code.
		self.insn_opid = 0
		# Hold opcode information, (opcode, opcode length, opcodeid, flags, [4 operand])
		self.insn_info = []
		self.opid_name = None
		self.grp_insn_info = {}

	def post_process(self):
		self.opid_name = [ (opid, name) for (name, opid) in self.name_opid.iteritems() ]
		self.opid_name.sort()

	def dump_insn_name(self):
		insn_list = [ '\t%#04x: "%s",\n' % (opid, name.lower()) for (opid, name) in self.opid_name ]
		dump = """// Use opcode id to index instruction mnemonics.
var InsnName = [...]string{
%s}
""" % ''.join(insn_list)
		return dump

	def dump_opcodeid(self):
		# We need to specify iota for the first opcode id
		l = ( "\tInsn_%s%s// %#04x\n" % (name.capitalize().replace(' ', '_'),
			len(name) <= 2 and ('\t' * 3) or ('\t' * 2), opid)  for opid, name in self.opid_name[1:] )
		return """const (
	Insn_%s byte = iota\t// 0x00
%s)
""" % (self.opid_name[0][1].capitalize().replace(' ', '_'), ''.join(l))

	# copied from x86header.py
	INSN_FLAGS =  """const (
	IFLAG_NONE = iota
	IFLAG_MODRM_REQUIRED = 1 << (iota - 1)
	IFLAG_NOT_DIVIDED
	IFLAG_16BITS
	IFLAG_32BITS
	IFLAG_PRE_LOCK
	IFLAG_PRE_REPNZ
	IFLAG_PRE_REP
	IFLAG_PRE_CS
	IFLAG_PRE_SS
	IFLAG_PRE_DS
	IFLAG_PRE_ES
	IFLAG_PRE_FS
	IFLAG_PRE_GS
	IFLAG_PRE_OP_SIZE
	IFLAG_PRE_ADDR_SIZE
	IFLAG_NATIVE
	IFLAG_USE_EXMNEMONIC
	IFLAG_USE_OP3
	IFLAG_USE_OP4
	IFLAG_MNEMONIC_MODRM_BASED
	IFLAG_MODRR_REQUIRED
	IFLAG_3DNOW_FETCH
	IFLAG_PSEUDO_OPCODE
	IFLAG_INVALID_64BITS
	IFLAG_64BITS
	IFLAG_PRE_REX
	IFLAG_USE_EXMNEMONIC2
	IFLAG_64BITS_FETCH
	IFLAG_FORCE_REG0
	IFLAG_PRE_VEX
	IFLAG_MODRM_INCLUDED
	IFLAG_DST_WR
	IFLAG_VEX_L
	IFLAG_VEX_W
	IFLAG_MNEMONIC_VEXW_BASED
	IFLAG_MNEMONIC_VEXL_BASED
	IFLAG_FORCE_VEXL
	IFLAG_MODRR_BASED
	IFLAG_VEX_V_UNUSED
)
"""

	def pos2key(self, pos):
		key = 0
		for (i, p) in enumerate(pos):
			key += p << (8 * i)
		return key

	# Different instructions in the same instruction group may have different
	# operands. To make handling of this easier, group and divided
	# instructions will have their instruction info stored in a map in the
	# generated Go file.
	def addToGrpInsnInfo(self, pos, reg, info):
		key = self.pos2key(list(reversed(pos + [reg])))
		self.grp_insn_info[key] = info

	def SetInstruction(self, *args):
		""" This function is used in order to insert an instruction info into the DB. """
		mnemonics = [a.lower() for a in args[2]]
		flags = args[4]
		operands = args[3]

		# *args = ISetClass, OL, pos, mnemonics, operands, flags
		# Construct an Instruction Info object with the info given in args.
		opcode = args[1].replace(" ", "").split(",")
		# The number of bytes is the base length, now we need to check the last entry.
		pos = [int(i[:2], 16) for i in opcode]

		# if len(self.insn_info):
		# 	print >>sys.stderr, pos, self.insn_info[-1][0]

		# Allocate new opcode id for new mnemonics
		for mn in mnemonics:
			if mn not in self.name_opid and mn != "":
				self.name_opid[mn] = self.insn_opid
				self.insn_opid += 1

		# Use the first mnemonics id if mnemonics is modrm based
		opcodeid = self.name_opid[mnemonics[0]]

		last = opcode[-1][2:] # Skip hex of last full byte
		isModRMIncluded = False # Indicates whether 3 bits of the REG field in the ModRM byte were used.
		if last[:2] == "//": # Divided Instruction
			reg = int(last[2:], 16)
			assert reg < 0xff
			isModRMIncluded = True
			try:
				OL = {1:OpcodeLength.OL_1d, 2:OpcodeLength.OL_2d}[len(opcode)]
			except KeyError:
				raise DBException("Invalid divided instruction opcode")
		elif last[:1] == "/": # Group Instruction
			isModRMIncluded = True
			reg = int(last[1:], 16)
			assert reg < 0xf
			try:
				OL = {1:OpcodeLength.OL_13, 2:OpcodeLength.OL_23, 3:OpcodeLength.OL_33}[len(opcode)]
			except KeyError:
				raise DBException("Invalid group instruction opcode")
		elif len(last) != 0:
			raise DBException("Invalid last byte in opcode")
			# Normal full bytes instruction
		else:
			try:
				OL = {1:OpcodeLength.OL_1, 2:OpcodeLength.OL_2, 3:OpcodeLength.OL_3, 4:OpcodeLength.OL_4}[len(opcode)]
			except KeyError:
				raise DBException("Invalid normal instruction opcode")

		insninfo = [pos, OL, opcodeid, flags, operands]

		# Store the instruction info in the grp insntruction specific map
		if isModRMIncluded:
			insninfo[3] |= InstFlag.MODRM_INCLUDED
			self.addToGrpInsnInfo(pos, reg, insninfo)

		# In case handling instruction with modrm included, we still need to
		# add (only one) InsnInfo in the 1st and 2nd InsnDB, so we know we
		# need to look up in the grpInsnInfo map to get the actual InsnInfo.
		if len(self.insn_info) > 0 and self.insn_info[-1][0] == pos:
			return
		self.insn_info.append(insninfo)

		# Generate all opcode for instructions which use the lowest 3 bits are reg field
		# Ugly code here ...
		for op in operands:
			if (op == OperandType.IB_RB) or (op == OperandType.IB_R_FULL):
				for i in xrange(1, 8):
					if len(opcode) == 1:
						self.insn_info.append(([pos[0] + i] + pos[1:], OL, opcodeid, flags, operands))
					elif len(opcode) == 2:
						self.insn_info.append(([pos[0], pos[1] + i] + pos[2:], OL, opcodeid, flags, operands))
				break

	OPERAND_TYPE = """const (
	OT_NONE byte = iota
	OT_IMM8
	OT_IMM16
	OT_IMM_FULL
	OT_IMM32
	OT_SEIMM8
	OT_IMM16_1
	OT_IMM8_1
	OT_IMM8_2
	OT_REG8
	OT_REG16
	OT_REG_FULL
	OT_REG32
	OT_REG32_64
	OT_FREG32_64_RM
	OT_RM8
	OT_RM16
	OT_RM_FULL
	OT_RM32_64
	OT_RM16_32
	OT_FPUM16
	OT_FPUM32
	OT_FPUM64
	OT_FPUM80
	OT_R32_M8
	OT_R32_M16
	OT_R32_64_M8
	OT_R32_64_M16
	OT_RFULL_M16
	OT_CREG
	OT_DREG
	OT_SREG
	OT_SEG
	OT_ACC8
	OT_ACC16
	OT_ACC_FULL
	OT_ACC_FULL_NOT64
	OT_MEM16_FULL
	OT_PTR16_FULL
	OT_MEM16_3264
	OT_RELCB
	OT_RELC_FULL
	OT_MEM
	OT_MEM_OPT
	OT_MEM32
	OT_MEM32_64
	OT_MEM64
	OT_MEM128
	OT_MEM64_128
	OT_MOFFS8
	OT_MOFFS_FULL
	OT_CONST1
	OT_REGCL
	OT_IB_RB
	OT_IB_R_FULL
	OT_REGI_ESI
	OT_REGI_EDI
	OT_REGI_EBXAL
	OT_REGI_EAX
	OT_REGDX
	OT_REGECX
	OT_FPU_SI
	OT_FPU_SSI
	OT_FPU_SIS
	OT_MM
	OT_MM_RM
	OT_MM32
	OT_MM64
	OT_XMM
	OT_XMM_RM
	OT_XMM16
	OT_XMM32
	OT_XMM64
	OT_XMM128
	OT_REGXMM0
	OT_RM32
	OT_REG32_64_M8
	OT_REG32_64_M16
	OT_WREG32_64
	OT_WRM32_64
	OT_WXMM32_64
	OT_VXMM
	OT_XMM_IMM
	OT_YXMM
	OT_YXMM_IMM
	OT_YMM
	OT_YMM256
	OT_VYMM
	OT_VYXMM
	OT_YXMM64_256
	OT_YXMM128_256
	OT_LXMM64_128
	OT_LMEM128_256
)
"""

	def dump_ot2size(self):
		otwithsize = []
		lines = self.OPERAND_TYPE.splitlines()
		for line in lines:
			line = line.strip()
			if not line.startswith("OT_"): continue
			if line.find('FULL') != -1:
				otwithsize.append(line + ': OpSizeFull,\n')
			elif line.find('IMM') != -1 or line.find('MEM') != -1 or line.find('REG') != -1 or \
				line.find('RM') != -1 or line.find('ACC') != -1:
				if line.find('128') != -1:
					continue
				elif line.find('8') != -1:
					otwithsize.append(line + ': OpSizeByte,\n')
				elif line.find('16') != -1:
					otwithsize.append(line + ': OpSizeWord,\n')
				elif line.find('32') != -1:
					otwithsize.append(line + ': OpSizeLong,\n')
		dump = """var ot2size = [%d]byte{
	%s
}
""" % (len(lines) - 2, '\t'.join(otwithsize))
		return dump

	def dump_1insn(self, pos, opcodeid, flag, operand):
		return '%#04x: InsnInfo{ %#04x, %#x, [4]byte{%s} },\n' % (pos, opcodeid, flag, ', '.join('%d' % i for i in operand))

	def dump_insninfo(self):
		insn_list = [] # table for the 1st byte of instruction
		insn_list2 = [] # table for the 2nd byte of instruction
		for (pos, OL, opcodeid, flag, operand) in self.insn_info:
			if OL in (OpcodeLength.OL_1, OpcodeLength.OL_13, OpcodeLength.OL_1d):
				insn_list.append(self.dump_1insn(pos[0], opcodeid, flag, operand))
			elif OL in (OpcodeLength.OL_2, OpcodeLength.OL_23, OpcodeLength.OL_2d):
				insn_list2.append(self.dump_1insn(pos[1], opcodeid, flag, operand))
			else:
				print pos
				raise DBException("Does not support instruction longer than 2 bytes")
				# s = '\t%#04x: InsnInfo{ %#04x, %#x, [4]byte{%s} },\n' % (pos[3], opcodeid, flag, ', '.join(['%d' % i for i in operand]))
				# insn_list2.append(s)

		dump = """// Opcode to instruction info map.
// Table for the 1st byte of instruction
var InsnDB = [...]InsnInfo{
	%s}

// Table for the 2nd byte of instruction
var InsnDB2 = [...]InsnInfo{
%s}
""" % ('\t'.join(insn_list), '\t'.join(insn_list2))
		return dump

	def dump_grp_insninfo(self):
		# Go's address operator has limitations. So here I use a map to get
		# the index into an array to get around that limitation. Don't need

		# the opcode in the dumped instruction info here is not needed
		insninfo_arr = [ (key, v) for (key, v) in self.grp_insn_info.iteritems() ]
		insninfo_arr.sort()
		idx = []
		infos = []
		for (i, (key, (pos, _, opcodeid, flag, operand))) in enumerate(insninfo_arr):
			if key >= 0xf0000:
				idx.append("\t%#08x: %d,\n" % (key, i))
			else:
				idx.append("\t%#06x: %d,\n" % (key, i))
			infos.append("\t%d: %s" % (i, self.dump_1insn(pos[0], opcodeid, flag, operand)[6:]))

		dump = """var grpInsnInfoIndex = map[int]int{
%s}

var grpInsnInfo = [...]InsnInfo{
%s}""" % (''.join(idx), ''.join(infos))
		return dump

	def dump(self):
		self.post_process()
		print 'package dis\n'
		print self.INSN_FLAGS
		print self.OPERAND_TYPE
		print self.dump_ot2size()
		print self.dump_opcodeid()
		print self.dump_insn_name()
		print self.dump_insninfo()
		print self.dump_grp_insninfo()

def main():
	db = InstructionDB()
	x86sets.Instructions(db.SetInstruction)

	db.dump()

main()