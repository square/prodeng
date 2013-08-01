all:
	gcc -Wall -Werror -Wstrict-prototypes -std=c99 -pedantic-errors nolimit.c -o nolimit

clean:
	rm -f *.o nolimit

test:
	sudo ./nolimit true
