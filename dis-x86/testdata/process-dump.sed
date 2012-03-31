s/^(jmp|jb|jne|je|call|loop|ja) .*$/\1 /
s/^nop$/nop /
s/^jne/jnz/
s/^je/jz/
s/^sete/setz/
s/^movzbl/movzx/
s/^movsbl/movsx/
