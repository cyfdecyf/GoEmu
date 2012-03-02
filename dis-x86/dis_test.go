package dis

import (
	"testing"
	"os"
)

type SliceReader []byte

func (buf SliceReader) ReadAt(p []byte, off int64) (n int, err os.Error) {
	var i int
	for i = 0; i < len(p) && i < (len(buf)-int(off)); i++ {
		p[i] = buf[int(off)+i]
	}
	return i, nil
}

func checkDump(dump string, expected string, t *testing.T) {
	if dump != expected {
		t.Errorf("expected: %s\tgot: %s\n", expected, dump)
	}
}

func TestPrefixParse(t *testing.T) {
	binary := SliceReader([]byte{0xf0, 0x88, 0x67})
	dc := NewDisContext(binary)

	dc.parsePrefix()
	if dc.Prefix&PrefixLOCK != PrefixLOCK {
		t.Error("Prefix lock not detected")
	}
	if dc.offset != 1 {
		t.Error("Offset should advance on correct Prefix")
	}

	dc.parsePrefix()
	if dc.offset != 1 {
		t.Error("Offset should not advance on non Prefix")
	}

	dc.offset++
	dc.parsePrefix()
	if dc.Prefix&PrefixAddressSize != PrefixAddressSize {
		t.Error("Prefix address size not detected")
	}
	if dc.Prefix&PrefixLOCK != PrefixLOCK {
		t.Error("Prefix lock should not be dropped")
	}
}

func TestArith(t *testing.T) {
	binary := SliceReader([]byte{
		0x03, 0x05, 0x01, 0x00, 0x00, 0x00, // add 0x1,%eax
		0x05, 0x32, 0x54, 0x12, 0x00, // add $0x125432,%eax
		0x03, 0x45, 0x08, // add 0x8(%ebp),%eax
		0x03, 0x04, 0x8d, 0x80, 0xa0, 0x2c, 0xc0, // add -0x3fd35f80(,%ecx,4),%eax
	})
	dc := NewDisContext(binary)

	dc.NextInsn()
	if dc.Opcode != OpAdd {
		t.Error("Add arithmetic insn not detected")
	}
	dump := dc.DumpInsn()
	checkDump(dump, "add 0x1,%eax", t)

	dc.NextInsn()
	dump = dc.DumpInsn()
	checkDump(dump, "add $0x125432,%eax", t)

	dc.NextInsn()
	dump = dc.DumpInsn()
	checkDump(dump, "add 0x8(%ebp),%eax", t)

	dc.NextInsn()
	dump = dc.DumpInsn()
	checkDump(dump, "add -0x3fd35f80(,%ecx,4),%eax", t)
}
