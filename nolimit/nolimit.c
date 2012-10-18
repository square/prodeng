/*
 * Derived from djb's daemontools.
 * See http://cr.yp.to/daemontools/
 *
 *
 */

#include <errno.h>
#include <sys/types.h>
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <pwd.h>
#include <string.h>
#include <strings.h>
#include <sys/resource.h>

#define PROG_NAME "nolimit"

void die(int retval, char * msg1, char * msg2) {
  fprintf(stderr, "%s%s\n", msg1, msg2);
  exit(retval);
}

/* Read a single integer from a file. Used for /proc */
long int int_from_file(const char const * filename) {
  char file_data[128];
  size_t bytes_read;
  FILE *fd;
  long int number;
  if ((fd = fopen(filename, "r")) == NULL) {
    die(111, PROG_NAME": Can't open file for reading", strerror(errno));
  }
  bytes_read = fread(file_data, 1, sizeof(file_data) - 1, fd);
  if (bytes_read == 0 && !feof(fd)) {
    die(111, PROG_NAME": read error: ", strerror(errno));
  } else if (bytes_read >= sizeof(file_data) - 1) {
    die(111, PROG_NAME": int_from_file: file too large: ", file_data);
  }
  errno = 0;
  number = strtol(file_data, NULL, 10);
  if (errno) {
    die(111, PROG_NAME": error, strol() failure reading file", strerror(errno));
  }
  return number;
}

int main(int argc, char *const argv[], const char *const *envp) {
  struct rlimit lim;
  argv++;
  bzero(&lim, sizeof(lim));

  /* RLIMIT_NOFILE cannot be set to infinity. Instead, discover its local maximal value and use that */

#ifdef __APPLE__
#include <sys/syslimits.h>
  lim.rlim_cur = OPEN_MAX;
  lim.rlim_max = OPEN_MAX;
  if (setrlimit(RLIMIT_NOFILE, &lim)) {
    die(111, "setlimit: fatal: can't setrlimit(RLIMIT_NOFILE,)", strerror(errno));
  }
#else
 #ifdef __linux__

/*  /proc/sys/fs/nr_open has the maximum value. from kernel/sys.c:
       if (resource == RLIMIT_NOFILE && new_rlim.rlim_max > sysctl_nr_open)
              return -EPERM;
*/

  long int nr_open = int_from_file("/proc/sys/fs/nr_open");
  lim.rlim_cur = nr_open;
  lim.rlim_max = nr_open;
  if (setrlimit(RLIMIT_NOFILE, &lim)) {
    die(111, "setlimit: fatal: can't setrlimit(RLIMIT_NOFILE,)", strerror(errno));
  }

 #endif
#endif

  /* These limits are easy - can all be infinity. */
  lim.rlim_cur = RLIM_INFINITY;
  lim.rlim_max = RLIM_INFINITY;
  /*
  Don't unlimit core by default.
  if (setrlimit(RLIMIT_CORE, &lim)) {
    die(111, "setlimit: fatal: can't setrlimit(RLIMIT_CORE,)", strerror(errno));
  }
  */
  if (setrlimit(RLIMIT_CPU, &lim)) {
    die(111, "setlimit: fatal: can't setrlimit(RLIMIT_CPU,)", strerror(errno));
  }
  if (setrlimit(RLIMIT_DATA, &lim)) {
    die(111, "setlimit: fatal: can't setrlimit(RLIMIT_DATA,)", strerror(errno));
  }
  if (setrlimit(RLIMIT_FSIZE, &lim)) {
    die(111, "setlimit: fatal: can't setrlimit(RLIMIT_FSIZE,)", strerror(errno));
  }
  if (setrlimit(RLIMIT_MEMLOCK, &lim)) {
    die(111, "setlimit: fatal: can't setrlimit(RLIMIT_MEMLOCK,)", strerror(errno));
  }
  if (setrlimit(RLIMIT_NPROC, &lim)) {
    die(111, "setlimit: fatal: can't setrlimit(RLIMIT_NPROC,)", strerror(errno));
  }
  if (setrlimit(RLIMIT_RSS, &lim)) {
    die(111, "setlimit: fatal: can't setrlimit(RLIMIT_RSS,)", strerror(errno));
  }
  /*
  Changing the stack size can change the way an app allocates memory. leave as-is
  lim.rlim_cur = 1024 * 1024 * 63;
  lim.rlim_max = 1024 * 1024 * 63;

  if (setrlimit(RLIMIT_STACK, &lim)) {
    die(111, "setlimit: fatal: can't setrlimit(RLIMIT_STACK,)", strerror(errno));
  }
  */
  execvp(argv[0], argv);
  die(111, "setlimit: fatal: unable to run ", argv[0]);
  return 1; /* not reached */
}
