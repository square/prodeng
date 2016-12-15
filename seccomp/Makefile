all: kafel/libkafel.a
	gcc -Wall -Werror  -Wstrict-prototypes -ansi -pedantic-errors -I./kafel/include/ seccomp.c kafel/libkafel.a -o seccomp
	gcc -Wall -Werror  -Wstrict-prototypes -std=c99 -pedantic-errors -I./kafel/include/ seccomp.c kafel/libkafel.a -o seccomp

kafel/libkafel.a:
	$(MAKE) -C kafel

clean:
	rm -f *.o seccomp
	$(MAKE) -C kafel clean

test:
	./seccomp -s "POLICY sample {}\nUSE sample DEFAULT ALLOW" true 
