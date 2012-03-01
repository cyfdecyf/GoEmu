package dis

import (
	"testing"
	"os"
	"fmt"
)

type SliceReader []byte

func (buf SliceReader) ReadAt(p []byte, off int64) (n int, err os.Error) {
	var i int
	for i = 0; i < len(p) && i < (len(buf)-int(off)); i++ {
		p[i] = buf[int(off)+i]
	}
	return i, nil
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
	if dc.Prefix&PrefixAddrSize != PrefixAddrSize {
		t.Error("Prefix address size not detected")
	}
	if dc.Prefix&PrefixLOCK != PrefixLOCK {
		t.Error("Prefix lock should not be dropped")
	}
}

func TestArith(t *testing.T) {
	// add 0x1,%eax
	binary := SliceReader([]byte{0x03, 0x05, 0x01, 0x00, 0x00, 0x00})
	dc := NewDisContext(binary)

	dc.NextInsn()
	if dc.Opcode != OpAdd {
		t.Error("Add arithmetic insn not detected")
	}
	dump := dc.DumpInsn()
	fmt.Println(dump)
}
