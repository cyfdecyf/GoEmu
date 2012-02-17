package dis

import (
	"testing"
	"os"
)

type SliceReader []byte

func (buf SliceReader) ReadAt(p []byte, off int64) (n int, err os.Error) {
	var i int
	for i = 0; i < len(p) && i < (len(buf) - int(off)); i++ {
		p[i] = buf[int(off) + i]
	}
	return i, nil
}

func TestPrefixParse(t *testing.T) {
	var dc Disassembler
	dc.binary = SliceReader([]byte{ 0xF0, 0x88, 0x67 })
	dc.offset = 0

	dc.parsePrefix()
	if dc.prefix & prefixLOCK != prefixLOCK {
		t.Error("Prefix lock not detected")
	}
	if dc.offset != 1 {
		t.Error("Offset should advance on correct prefix")
	}

	dc.parsePrefix()
	if dc.offset != 1 {
		t.Error("Offset should not advance on non prefix")
	}

	dc.offset++
	dc.parsePrefix()
	if dc.prefix & prefixADDR != prefixADDR {
		t.Error("Prefix address size not detected")
	}
	if dc.prefix & prefixLOCK != prefixLOCK {
		t.Error("Prefix lock should not be dropped")
	}
}
