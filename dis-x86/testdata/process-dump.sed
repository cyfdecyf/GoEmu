s/^(jmp|jb|jne|je|call|loop|ja|jbe) [^*]*$/\1 /
s/^nop$/nop /
s/^jne/jnz/
s/^je/jz/
s/^setne/setnz/
s/^sete/setz/
s/^ud2a $/ud2 /
