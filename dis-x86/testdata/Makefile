CC = gcc
CFLAGS = -O2 -m32
OBJS = add.o

all: $(OBJS)

%.o: %.c
	$(CC) -c $(CFLAGS) $<

clean:
	rm -f $(OBJS)
