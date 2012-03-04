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
	if i < len(p) {
		return i, os.EOF
	}
	return i, nil
}

func checkDump(dc *DisContext, expected string, t *testing.T) {
	if dc == nil {
		if expected != "" {
			t.Errorf("EOF not handled correctly\n") 
		}
		return
	}
	dump := dc.DumpInsn()
	if dump != expected {
		t.Errorf("expect: %s\nget:    %s\n", expected, dump)
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
		0x00, 0x00, // add %al,(%eax)
		0x00, 0x45, 0xf3, // add %al,-0xd(%ebp)
		0x02, 0x54, 0x28, 0xe5, // add -0x1b(%eax,%ebp,1),%dl
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
	checkDump(dc, "add %al,(%eax)", t)
	checkDump(dc.NextInsn(), "add %al,-0xd(%ebp)", t)
	checkDump(dc.NextInsn(), "add -0x1b(%eax,%ebp,1),%dl", t)
	checkDump(dc.NextInsn(), "add 0x1,%eax", t)
	checkDump(dc.NextInsn(), "add $0x125432,%eax", t)
	checkDump(dc.NextInsn(), "add 0x8(%ebp),%eax", t)
	checkDump(dc.NextInsn(), "add -0x3fd35f80(,%ecx,4),%eax", t)
}

func TestIncDec(t *testing.T) {
	binary := SliceReader([]byte{
		0x40, // inc %eax
		0x48, // dec %eax
		0x46, // inc %esi
	})
	dc := NewDisContext(binary)
	checkDump(dc.NextInsn(), "inc %eax", t)
	checkDump(dc.NextInsn(), "dec %eax", t)
	checkDump(dc.NextInsn(), "inc %esi", t)
}

func TestPushPop(t *testing.T) {
	binary := SliceReader([]byte{
		0x56, // push %esi
		0x57, // push %edi
		0x5d, // pop %ebp
		0x5b, // pop %ebx
		0x16, // push %ss
		0x1e, // push %ds
		0x1f, // pop %ds
	})
	dc := NewDisContext(binary)
	checkDump(dc.NextInsn(), "push %esi", t)
	checkDump(dc.NextInsn(), "push %edi", t)
	checkDump(dc.NextInsn(), "pop %ebp", t)
	checkDump(dc.NextInsn(), "pop %ebx", t)
	checkDump(dc.NextInsn(), "push %ss", t)
	checkDump(dc.NextInsn(), "push %ds", t)
	checkDump(dc.NextInsn(), "pop %ds", t)
	checkDump(dc.NextInsn(), "", t)
}

func TestMov(t *testing.T) {
	binary := SliceReader([]byte{
		0xb0, 0xeb, // mov $0xeb,%al
		0xb9, 0x2f, 0x00, 0x00, 0x00, // mov $0x2f,%ecx
		0xa0, 0x60, 0x96, 0x2c, 0xc0, // mov 0xc02c9660,%al
		0xa1, 0x9c, 0xf6, 0x2b, 0xc0, // mov 0xc02bf69c,%eax
		0xa3, 0x24, 0x01, 0x31, 0xc0, // mov %eax,0xc0310124
		0x89, 0xd8, // mov %ebx,%eax
		0x8a, 0x45, 0xec, // mov -0x14(%ebp),%al
		0x8c, 0xd0, // mov %ss,%eax
		0x8e, 0xd8, // mov %eax,%ds
		0x8e, 0xd9, // mov %ecx,%ds
	})
	dc := NewDisContext(binary)
	checkDump(dc.NextInsn(), "mov $0xeb,%al", t)
	checkDump(dc.NextInsn(), "mov $0x2f,%ecx", t)
	checkDump(dc.NextInsn(), "mov 0xc02c9660,%al", t)
	checkDump(dc.NextInsn(), "mov 0xc02bf69c,%eax", t)
	checkDump(dc.NextInsn(), "mov %eax,0xc0310124", t)
	checkDump(dc.NextInsn(), "mov %ebx,%eax", t)
	checkDump(dc.NextInsn(), "mov -0x14(%ebp),%al", t)
	checkDump(dc.NextInsn(), "mov %ss,%eax", t)
	checkDump(dc.NextInsn(), "mov %eax,%ds", t)
	checkDump(dc.NextInsn(), "mov %ecx,%ds", t)
}
