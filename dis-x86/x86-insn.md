Documents
=========

The 7 volumes Intel Manual. I'm using the version released in Dec. 2011.
For simplicity, I will call them Vol1, Vol2A, Vol2B etc.

Other onlines resources maybe easy to get started.

- [X86-64 Instruction Encoding](http://wiki.osdev.org/X86-64_Instruction_Encoding)
- [x86-64 Tour of Intel Manuals](http://x86asm.net/articles/x86-64-tour-of-intel-manuals/)

Operand size and address-size attribute
=======================================

Refer to Vol1 3.6. This note only talks about 32-bit case.

> In protected mode, every code segment has a default operand-size attribute and
> address-size attribute.

These attributes are selected by the D flag in segment descriptor.

- Protected mode

 D flag |  operand-size | address-size
--------|---------------|--------------
   1    |      32-bit   |     32-bit
   0    |      16-bit   |     16-bit

- In Real-address, virtual-8086 or SMM mode, both attriubtes are always 16-bits.

Source and dstination operands include:

- immediate operand (only in source operand)
- register
- memory location
- I/O port

The address-size attribute selects the sizes of addresses used to address memory.
It affects segment offset and displacement.

address-size  | segment offset | displacement
--------------|----------------|--------------
  16-bit      |     16-bit     |    16-bit
  32-bit      |     32-bit     |    32-bit

Prefix
======

- **Operand-size override prefix** can overide the operand and address size for
  a particular instruction.

  > The operand-size override prefix allows a program to switch between 16- and
  > 32-bit operand sizes. Either size can be the default; use of the prefix
  > selects the non-default size.

Opcode
======

Refer Vol2A Section 3.1.1 for how to interpret the instruction
reference.

Instruction formats and encoding
--------------------------------

Vol2C Appendix B Section B.2 Table B-13 lists all the instruction encoding
with all the special fields. We can use this table to generate all the
possible instruction encodings. (Maybe diStorm's trie is built using this
table.)

The "A", "B" superscripts for the "mod" in Table B-13 means specific encoding
of the mod field in ModR/M byte is reserved. Refer to B.1.5 Table B-12.

Opcode map
----------

Appendix A, B in Vol2C are very useful.

A.3 is the complete opcode map for 1, 2, 3 byte opcodes. Understanding that
opcode map needs some knowledge on the abbreviations used.

From A.2 KEY TO ABBREVIATIONS

> Operands are identified by a two-character code of the form Zz. The first
> character, an uppercase letter, specifies the addressing method; the second
> character, a lower-case letter, specifies the type of operand.

From A.2.1 Codes for Addressing Method

- Not requiring ModR/M
  - `A` -- Direct address: address of the operand is encoded in the instruction.
  - `O` -- The offset of the operand is coded as a word or double word
    (depending on address size attribute) in the instruction.

- Requiring ModR/M
  - `G` -- reg field selects a general register
  - `E` -- operand is either a general-purpose register or a memory address
  - `R` -- R/M field refer only to a general register
  - `M` -- ModR/M byte may refer only to memory
  - `I` -- Immediate data: the operand value is encoded in subsequent bytes of the instruction

From A.2.2 Codes for Operand Type

- `v` - word, doubleword or quadword, depending on operand-size attribute.
- `z` - For 16-bit operand-size, word; otherwise, doubleword

From A.2.3 Register Codes

- `eXX` for 16 or 32-bit size, eg. eAX can be AX or EAX
- `rXX` for 16, 32 or 64-bit size

What's Group 1?
---------------

In Table A-2, there's an "Immediate Grp 1(1A)" that does not specify what the
opcode actually do. The superscript symbol "1A" means (refer to Table A-1):

> Bits 5, 4, and 3 of ModR/M byte used as an opcode extension

So what the opcode does depends on the extension in the reg field of ModR/M byte.
