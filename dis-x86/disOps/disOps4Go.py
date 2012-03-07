import x86sets
from x86header import *

"""
Generate instruction definition for GoEmu
"""

class InstructionDB():
	def __init__(self):
		# Holds all the instructions, { "name" : opcodeId }
		self.all_insn = {}
		# Current largest instruction id
		self.insn_opid = 0

	def dump_insn(self):
		insn_arr = [ (opid, name) for (name, opid) in self.all_insn.iteritems() ]
		insn_arr.sort()
		insn_list = [ '\t%#04x: "%s",\n' % (opid, name.lower()) for (opid, name) in insn_arr ]
		dump = """var InsnName = [...]string{
%s}
""" % ''.join(insn_list)
		return dump

	# copied from x86header.py
	INSN_FLAGS =  """const (
	IFLAG_NONE = iota
	IFLAG_MODRM_REQUIRED
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

	# (iset-class, opcode-length, list of bytes of opcode, list of string of mnemonics, list of operands, flags) """
	def SetInstruction(self, *args):
		# ignore insn_class, and I use only the first mnemonics in all possible ones
		mnemonics = args[2][0].lower()
		if mnemonics not in self.all_insn:
			self.all_insn[mnemonics] = self.insn_opid
			self.insn_opid += 1
		pass

	def dump(self):
		print self.INSN_FLAGS
		print self.dump_insn()

def main():
	db = InstructionDB()
	x86sets.Instructions(db.SetInstruction)

	db.dump()

main()