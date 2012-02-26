package dis

import (
	"testing"
	"os"
	"fmt"
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
	var dc DisContext
	dc.binary = SliceReader([]byte{ 0xf0, 0x88, 0x67 })
	dc.offset = 0

	dc.parsePrefix()
	if dc.Prefix & PrefixLOCK != PrefixLOCK {
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
	if dc.Prefix & PrefixADDR != PrefixADDR {
		t.Error("Prefix address size not detected")
	}
	if dc.Prefix & PrefixLOCK != PrefixLOCK {
		t.Error("Prefix lock should not be dropped")
	}
}

func TestDisAddFunc(t *testing.T) {
	var dc DisContext
	// push, pop, ret
	dc.binary = SliceReader([]byte{ 0x55, 0x5d, 0xc3 })
	dc.offset = 0

	dc.NextInsn()
	if dc.Raw[0] != 0x55 {
		t.Error("Raw binary is not copied correctly")
	}
	if dc.Opcode != OpPush {
		t.Error("Push not recognised")
	}

	dc.NextInsn()
	if dc.Raw[0] != 0x5d {
		t.Error("Raw binary is not copied correctly")
	}
	if dc.Opcode != OpPop {
		t.Error("Pop not recognised")
	}

	dc.NextInsn()
	if dc.Raw[0] != 0xc3 {
		t.Error("Raw binary is not copied correctly")
	}
	if dc.Opcode != OpRet {
		t.Error("Pop not recognised")
	}
}

func TestDumpMove(t *testing.T) {
	var dc DisContext
	// mov %esp,%ebp
	dc.binary = SliceReader([]byte{ 0x89, 0xe5 })
	dc.offset = 0

	dc.NextInsn()

	if dc.Opcode != OpMov {
		t.Error("Opcode Mov not recognised")
	}

	// fmt.Printf("ModRM: 0x%x\n", dc.Raw[1])
	if dc.Mod != 0x3 {
		t.Errorf("Mod not correct, get 0x%x", dc.Mod)
	}

	dump := DumpInsn(&dc.Instrucion)
	fmt.Println(dump)
	if dump != "mov %esp,%ebp" {
		t.Errorf("Mov dump not correct, get: %s", dump)
	}
}

func TestDumpInc(t *testing.T) {
	var dc DisContext
	// mov %esp,%ebp
	dc.binary = SliceReader([]byte{ 0x40 })
	dc.offset = 0

	dc.NextInsn()
	if dc.Opcode != OpInc {
		t.Error("Opcode Inc not recognised")
	}

	dump := DumpInsn(&dc.Instrucion)
	fmt.Println(dump)
	if dump != "inc %eax" {
		t.Errorf("Inc dump not correct, get: %s", dump)
	}
}