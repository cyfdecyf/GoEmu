include $(GOROOT)/src/Make.inc

TARG=github.com/cyfdecyf/goemu/dis

GOFILES=\
	dis.go\
	prefix.go\
	opcode.go\
	dump.go

include $(GOROOT)/src/Make.pkg

GOFMT=gofmt
fmt:
	${GOFMT} -w ${GOFILES} dis_test.go
