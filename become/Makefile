all:
	gcc -Wall -Werror  -Wstrict-prototypes -ansi -pedantic-errors become.c -o become
	gcc -Wall -Werror  -Wstrict-prototypes -std=c99 -pedantic-errors become.c -o become

clean:
	rm -f *.o become

test:
	sudo ./become nobody true 
