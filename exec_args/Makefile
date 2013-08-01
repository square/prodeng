all:
	gcc -Wall -Werror  -Wstrict-prototypes -ansi -pedantic-errors exec_args.c -o exec_args
	gcc -Wall -Werror  -Wstrict-prototypes -std=c99 -pedantic-errors exec_args.c -o exec_args

clean:
	rm -f *.o exec_args

test:
	./exec_args true
