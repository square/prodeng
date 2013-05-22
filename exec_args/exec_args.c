/*
 * Exec argv. No-op, unless the executable is granted permissions when it runs.
 */

#include <stdlib.h>
#include <stdio.h>
#include <unistd.h>

void die(int retval, char * msg1, char * msg2) {
  fprintf(stderr, "%s%s\n", msg1, msg2);
  exit(retval);
}

int main(int argc, char *const argv[], const char *const *envp) {
  /* do we have at least one arg? Don't count our own executable name*/
  if (!*++argv) {
    die(100, "usage: exec_args argv", "");
  }

  execvp(argv[0], argv);
  die(111, "exec_args: fatal: unable to run ", argv[0]);
  return 1; /* not reached */
}
