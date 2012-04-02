package dis

import (
	"bufio"
	"debug/elf"
	"fmt"
	"io"
	"os"
	"testing"
)

type SliceReader []byte

// Any better name?
type codeText struct {
	binary   []byte
	assembly string
}

func codeTextArr2ReaderAt(ctdata []codeText) SliceReader {
	s := make([]byte, 0)
	for _, ct := range ctdata {
		s = append(s, ct.binary...)
	}
	return SliceReader(s)
}

func (buf SliceReader) ReadAt(p []byte, off int64) (n int, err error) {
	var i int
	for i = 0; i < len(p) && i < (len(buf)-int(off)); i++ {
		p[i] = buf[int(off)+i]
	}
	if i < len(p) {
		return i, io.EOF
	}
	return i, nil
}

func (dc *DisContext) dumpInsnBinary() (dump string) {
	bin := dc.binary.(SliceReader)
	for i := dc.insnStart; i < dc.offset; i++ {
		dump += fmt.Sprintf("%02x ", bin[i])
	}
	return
}

func checkDump1(dc *DisContext, expected string, t *testing.T) bool {
	if dc == nil {
		if expected != "" {
			t.Fatal("EOF not handled correctly")
		}
		return true
	}
	dump := dc.DumpInsn()
	if dump != expected {
		t.Logf("\nbinary: %s\nexpect: %s\nget:    %s\n",
			dc.dumpInsnBinary(), expected, dump)
		return false
	}
	return true
}

func checkDump(dc *DisContext, expected string, t *testing.T) {
	if !checkDump1(dc, expected, t) {
		t.FailNow()
	}
}

func testDump(testdata []codeText, t *testing.T) {
	dc := NewDisContext(codeTextArr2ReaderAt(testdata))
	for _, ct := range testdata {
		checkDump(dc.NextInsn(), ct.assembly, t)
	}
	checkDump(dc.NextInsn(), "", t)
}

func TestPrefixParse(t *testing.T) {
	binary := SliceReader([]byte{0xf0, 0x88, 0x67, 0x89}) // Add one more byte to avoid EOF
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

	// Skip the non prefix byte
	if dc.nextByte() != 0x88 {
		t.Error("Byte not correct after parsing non prefix.")
	}

	dc.parsePrefix()
	if dc.offset != 3 {
		t.Error("Offset not correct when parsing prefix")
	}
	if dc.Prefix&PrefixAddressSize != PrefixAddressSize {
		t.Error("Prefix address size not detected")
	}
	if dc.Prefix&PrefixLOCK != PrefixLOCK {
		t.Error("Prefix lock should not be dropped")
	}

	testdata := []codeText{
		codeText{[]byte{0x64, 0x8b, 0x35, 0x40, 0xce, 0x2f, 0xc0}, "mov %fs:0xc02fce40,%esi"},
		codeText{[]byte{0xf0, 0x83, 0x04, 0x24, 0x00}, "lock addl $0x0,(%esp)"},
		codeText{[]byte{0xf3, 0xab}, "rep stos %eax,%es:(%edi)"},
	}
	testDump(testdata, t)
}

func TestArith(t *testing.T) {
	testdata := []codeText{
		codeText{[]byte{0x00, 0x00}, "add %al,(%eax)"},
		codeText{[]byte{0x00, 0x45, 0xf3}, "add %al,-0xd(%ebp)"},
		codeText{[]byte{0x02, 0x54, 0x28, 0xe5}, "add -0x1b(%eax,%ebp,1),%dl"},
		codeText{[]byte{0x03, 0x05, 0x01, 0x00, 0x00, 0x00}, "add 0x1,%eax"},
		codeText{[]byte{0x05, 0x32, 0x54, 0x12, 0x00}, "add $0x125432,%eax"},
		codeText{[]byte{0x03, 0x45, 0x08}, "add 0x8(%ebp),%eax"},
		codeText{[]byte{0x03, 0x04, 0x8d, 0x80, 0xa0, 0x2c, 0xc0}, "add -0x3fd35f80(,%ecx,4),%eax"},
	}
	testDump(testdata, t)
}

func TestIncDec(t *testing.T) {
	testdata := []codeText{
		codeText{[]byte{0x40}, "inc %eax"},
		codeText{[]byte{0x48}, "dec %eax"},
		codeText{[]byte{0x46}, "inc %esi"},
	}
	testDump(testdata, t)
}

func TestPushPop(t *testing.T) {
	testdata := []codeText{
		codeText{[]byte{0x56}, "push %esi"},
		codeText{[]byte{0x57}, "push %edi"},
		codeText{[]byte{0x5d}, "pop %ebp"},
		codeText{[]byte{0x5b}, "pop %ebx"},
		codeText{[]byte{0x16}, "push %ss"},
		codeText{[]byte{0x1e}, "push %ds"},
		codeText{[]byte{0x1f}, "pop %ds"},
	}
	testDump(testdata, t)
}

func TestMov(t *testing.T) {
	testdata := []codeText{
		codeText{[]byte{0x64, 0xa1, 0x40, 0xce, 0x2f, 0xc0}, "mov %fs:0xc02fce40,%eax"},
		codeText{[]byte{0x8b, 0x1d, 0xa8, 0x6b, 0x25, 0xc0}, "mov 0xc0256ba8,%ebx"},
		codeText{[]byte{0x88, 0x44, 0x3d, 0xd8}, "mov %al,-0x28(%ebp,%edi,1)"},
		codeText{[]byte{0xb0, 0xeb}, "mov $0xeb,%al"},
		codeText{[]byte{0xb9, 0x2f, 0x00, 0x00, 0x00}, "mov $0x2f,%ecx"},
		codeText{[]byte{0xa0, 0x60, 0x96, 0x2c, 0xc0}, "mov 0xc02c9660,%al"},
		codeText{[]byte{0xa1, 0x9c, 0xf6, 0x2b, 0xc0}, "mov 0xc02bf69c,%eax"},
		codeText{[]byte{0xa3, 0x24, 0x01, 0x31, 0xc0}, "mov %eax,0xc0310124"},
		codeText{[]byte{0x89, 0xd8}, "mov %ebx,%eax"},
		codeText{[]byte{0x8a, 0x45, 0xec}, "mov -0x14(%ebp),%al"},
		codeText{[]byte{0x8c, 0xd0}, "mov %ss,%eax"},
		codeText{[]byte{0x8e, 0xd8}, "mov %eax,%ds"},
		codeText{[]byte{0x8e, 0xd9}, "mov %ecx,%ds"},
	}
	testDump(testdata, t)
}

func TestTest(t *testing.T) {
	testdata := []codeText{
		codeText{[]byte{0xf6, 0x86, 0x11, 0x02, 0x00, 0x00, 0x40}, "testb $0x40,0x211(%esi)"},
	}
	testDump(testdata, t)
}

func TestJcc(t *testing.T) {
	testdata := []codeText{
		codeText{[]byte{0x75, 0x16}, "jnz "},
	}
	testDump(testdata, t)
}

func TestLgdt(t *testing.T) {
	testdata := []codeText{
		codeText{[]byte{0x0f, 0x01, 0x15, 0xd2, 0xcd, 0x2b, 0x00}, "lgdtl 0x2bcdd2"},
	}
	testDump(testdata, t)
}

func TestLea(t *testing.T) {
	testdata := []codeText{
		codeText{[]byte{0x8d, 0xa1, 0x00, 0x00, 0x00, 0x40}, "lea 0x40000000(%ecx),%esp"},
	}
	testDump(testdata, t)
}

func TestNop(t *testing.T) {
	testdata := []codeText{
		codeText{[]byte{0x90}, "nop "},
		codeText{[]byte{0x66, 0x90}, "xchg %ax,%ax"},
	}
	testDump(testdata, t)
}

func TestCall(t *testing.T) {
	testdata := []codeText{
		codeText{[]byte{0xe8, 0x52, 0x9e, 0x0f, 0x00}, "call "},
		codeText{[]byte{0xff, 0x15, 0x5c, 0xb7, 0x30, 0xc0}, "call *0xc030b75c"},
	}
	testDump(testdata, t)
}

// Disassemble the Linux kernel vmlinux file, see if the result matches
// objdump's output.
func checkLinux(t *testing.T) {
	// Open the dump result
	df, e := os.Open("testdata/dump.vmlinux")
	if e != nil {
		t.Fatal("open dump.linux failed")
	}
	dumpReader := bufio.NewReaderSize(df, 256)

	// Open vmlinux
	f, e1 := elf.Open("testdata/vmlinux")
	if e1 != nil {
		t.Fatal("open vmlinux failed")
	}
	textSection := f.Section(".text")
	if textSection == nil {
		t.Fatal("finding text section failed")
	}
	rawbytes, e2 := textSection.Data()
	if e2 != nil {
		t.Fatal("reading text section failed")
	}
	dc := NewDisContext(SliceReader(rawbytes))

	// Test instruction dump one by one
	for i := 1; ; i++ {
		line, isPrefix, err := dumpReader.ReadLine()
		if isPrefix {
			t.Fatal("Disassemble file has very long line")
		}
		if !checkDump1(dc.NextInsn(), string(line), t) {
			t.Fatal("Failed parsing the", i, "instruction")
		}

		if err == io.EOF {
			break
		} else if err != nil {
			t.Fatal("Disassemble file read error:", err)
		}
	}
}

func TestLinuxKernel(t *testing.T) {
	checkLinux(t)
}
