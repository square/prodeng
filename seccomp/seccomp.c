#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <getopt.h>

#include <linux/filter.h>
#include <linux/seccomp.h>
#include <sys/prctl.h>

#include "kafel.h"

void die(int retval, char * msg1, char * msg2) {
  fprintf(stderr, "%s%s\n", msg1, msg2);
  exit(retval);
}

int main(int argc, char *argv[])
{
  int opt;
  char *kafelFile = NULL;
  char *kafelString = NULL;
  struct sock_fprog seccompFilter;
  kafel_ctxt_t kafelContext;

  while ((opt = getopt(argc, argv, "f:s:")) != -1) {
    switch (opt) {
    case 'f': kafelFile = optarg; break;
    case 's': kafelString = optarg; break;
    default:
      die(100, "seccomp: usage: seccomp [-fs] argv", "");
    }
  }

  /* Move the argv pointer forward so that we can exec it */
  argv += optind;

  if (!*argv) {
    die(100, "seccomp: usage: seccomp [-fs] argv", "");
  }
  
  if (kafelFile == NULL && kafelString == NULL) {
    die(100, "seccomp: usage: seccomp [-fs] argv", "either -f or -s are required");
  }    

  kafelContext = kafel_ctxt_create();

  if (kafelFile != NULL) {
    FILE *kafelFileDescriptor = NULL;
    if ((kafelFileDescriptor = fopen(optarg, "r")) == NULL) {
      die(111, "seccomp: fatal: could not open", kafelFile);
    }
    kafel_set_input_file(kafelContext, kafelFileDescriptor);
  } else {
    kafel_set_input_string(kafelContext, kafelString);
  }

  if (kafel_compile(kafelContext, &seccompFilter) != 0) {
    die(111, "seccomp: fatal: could not compile policy", (char *)kafel_error_msg(kafelContext));
  }
  kafel_ctxt_destroy(&kafelContext);

  if (prctl(PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0)) {
    die(111, "seccomp: fatal: prctl(PR_SET_NO_NEW_PRIVS, 1) failed", "");
  }
  if (prctl(PR_SET_SECCOMP, SECCOMP_MODE_FILTER, &seccompFilter, 0, 0)) {
    die(111, "seccomp: fatal: prctl(PR_SET_SECCOMP, SECCOMP_MODE_FILTER) failed", "");
  }
  
  execvp(argv[0], argv);
  die(111, "seccomp: fatal: unable to run ", argv[0]);
  return 1; /* not reached */
}
