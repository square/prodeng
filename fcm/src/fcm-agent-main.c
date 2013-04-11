/*
 * A C implementation of the perl fcm-agent
 */

#include "fcm.h"

int main(int argc, char *argv[])
{
  // Let's talk variables!
  fcm_opts_t *opts = NULL;
  struct stat ad;
  struct stat dd;
  apr_pool_t *pool;
  apr_file_t *out;
  apr_file_t *err;
  int ret;

  // Initialize apr
  apr_initialize();
  atexit(apr_terminate);
  // delicious memory pool
  apr_pool_create(&pool, NULL);
  // Set up printing to the console
  apr_file_open_stdout(&out, pool);
  apr_file_open_stderr(&err, pool);

  //get our opts going
  opts = apr_palloc(pool, sizeof(fcm_opts_t));

  // Set up some defaults that we will blow away with command line args
  // if need be
  opts->agent_dir = apr_pstrdup(pool, "/var/fcm/agent");
  opts->data_dir = apr_pstrdup(pool, "/var/fcm/data");
  opts->sleep_time = 600;
  opts->verbose = 0;
  opts->run_once = 0;
  opts->out = out;
  opts->err = err;
  opts->pool = pool;

  ret = fcm_parse_opts(opts, argc, argv);
  if (ret != 0) return ret;

  if(stat(opts->agent_dir, &ad) != 0 || ! S_ISDIR(ad.st_mode))
  {
    apr_file_printf(err, "Agent directory does not exist: %s\n", opts->agent_dir);
    return 1;
  }
  else if(stat(opts->data_dir, &dd) != 0 || ! S_ISDIR(ad.st_mode))
  {
    apr_file_printf(err, "Data directory does not exist: %s\n", opts->data_dir);
    return 1;
  }

  agent_loop(opts);

  return 0;
}

