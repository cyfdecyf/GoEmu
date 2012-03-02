void foo() {
    asm volatile ("add 1,%eax");
}

int bar(int b) {
    b = b + 0x125432;
    return b;
}
