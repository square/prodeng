/*
 * Derived from the public domain setuidgid from djb's daemontools.
 * See http://cr.yp.to/daemontools/setuidgid.html
 *
 * Behavior is very, very similar. Major differences:
 *  * More meaningful error reporting
 *  * Sets supplementary groups
 *
 */

#define _BSD_SOURCE
#include <errno.h>
#include <grp.h>
#include <sys/types.h>
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>
#include <pwd.h>
#include <string.h>

void die(int retval, char * msg1, char * msg2) {
  fprintf(stderr, "%s%s\n", msg1, msg2);
  exit(retval);
}

int main(int argc, char *const argv[], const char *const *envp) {
  const char *username;
  struct passwd *pw;
  username = *++argv; /* shift first arg as username */
  if (!username || !*++argv) {
    die(100, "become: usage: become user argv", "");
  }
  pw = getpwnam(username);
  if (!pw) {
    die(111, "become: fatal: unknown user", (char *)username);
  }
  if (initgroups(username, pw->pw_gid)) {
    if (geteuid()) {
      die(111, "become: fatal: unable to initgroups, invoking user is not root: ", strerror(errno));
    }
    die(111, "become: fatal: unable to initgroups: ", strerror(errno));
  }
  if (setgid(pw->pw_gid)) {
    die(111, "become: fatal: unable to setgid: ", strerror(errno));
  }
  if (setuid(pw->pw_uid)) {
    die(111, "become: fatal: unable to setuid: ", strerror(errno));
  }
  execvp(argv[0], argv);
  die(111, "become: fatal: unable to run ", argv[0]);
  return 1; /* not reached */
}
