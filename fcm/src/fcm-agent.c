/*
 * A C implementation of the perl fcm-agent
 */

#include "fcm.h"

int fcm_parse_opts(fcm_opts_t *opts, int argc, char *argv[]) {
  int c;
  char full_path[PATH_MAX];

  // Args
  // -a = agent dir
  // -d = data dir
  // -h = help
  // -o = run once
  // -s = sleep time
  // -v = verbose
  //
  while ((c = getopt (argc, argv, "a:d:hos:v")) != -1)
  {
    switch (c)
    {
      case 'a':
        realpath(optarg, full_path);
        opts->agent_dir = apr_pmemdup(opts->pool, full_path, sizeof(full_path));
        break;
      case 'd':
        realpath(optarg, full_path);
        opts->data_dir = apr_pmemdup(opts->pool, full_path, sizeof(full_path));
        break;
      case 'h':
        apr_file_printf(opts->out, "Usage: fcm-agent [adhsv]\n");
        apr_file_printf(opts->out, "-a: Agent directory\n");
        apr_file_printf(opts->out, "-d: Data directory\n");
        apr_file_printf(opts->out, "-s: sleep time between runs\n");
        apr_file_printf(opts->out, "-v: print verbose messages\n");
        apr_file_printf(opts->out, "-h: this help message\n");
        return 1;
      case 'o':
        opts->run_once = 1;
        break;
      case 's':
        opts->sleep_time = atoi(optarg);
        break;
      case 'v':
        opts->verbose = 1;
        break;
      case '?':
        if (optopt == 'a' || optopt == 'd' || optopt == 's')
          apr_file_printf(opts->err, "Option -%a requires and argument.\n", optopt);
        else if (isprint (optopt))
          apr_file_printf(opts->err, "Unknown option -%c\n", optopt);
        else
          apr_file_printf(opts->err, "Unknown options character \\x%x\n", optopt);
        return 1;
      default:
        abort();
    }
  }
  return 0;
}

void agent_loop(fcm_opts_t *opts)
{

  DIR *dir_h;
  struct dirent *file;
  struct stat *af;
  struct stat *df;
  char *agent_file = NULL;
  char *data_file = NULL;
  char *module_name = NULL;
  char *pid_string = NULL;
  char *cpid_string = NULL;
  apr_hash_t *pid_map = NULL;
  pid_t *cpid;
  int *mod_pid;
  int *cstatus;


  while(1)
  {
    apr_pool_t *subpool;
    apr_pool_create(&subpool, opts->pool);

    dir_h = apr_palloc(subpool, sizeof(dir_h));
    file = apr_palloc(subpool, sizeof(struct dirent));
    af = apr_palloc(subpool, sizeof(struct stat));
    df = apr_palloc(subpool, sizeof(struct stat));
    agent_file = apr_palloc(subpool, PATH_MAX);
    data_file = apr_palloc(subpool, PATH_MAX);
    module_name = apr_palloc(subpool, PATH_MAX);
    pid_string = apr_palloc(subpool, 8);
    cpid_string = apr_palloc(subpool, 8);
    pid_map = apr_hash_make(subpool);
    cpid = apr_palloc(subpool, sizeof(pid_t));
    mod_pid = apr_palloc(subpool, sizeof(int));
    cstatus = apr_palloc(subpool, sizeof(int));

    apr_file_printf(opts->out, "\nWaking up for run\n");
    dir_h = opendir(opts->agent_dir);
    while (dir_h)
    {
      if ((file = readdir(dir_h)) != NULL)
      {
        if (apr_strnatcmp(file->d_name, ".") == 0 || apr_strnatcmp(file->d_name, "..") == 0)
        {
          if (opts->verbose) apr_file_printf(opts->out, "Skipping: %s/%s\n", opts->agent_dir, file->d_name);
          continue;
        }
        else
        {
          if (opts->verbose) apr_file_printf(opts->out, "Found: %s/%s\n", opts->agent_dir, file->d_name);
          agent_file = apr_psprintf(subpool, "%s/%s", opts->agent_dir, file->d_name);
          data_file = apr_psprintf(subpool, "%s/%s", opts->data_dir, file->d_name);
          if (stat(agent_file, af) == 0 && S_ISREG(af->st_mode) && stat(data_file, df) == 0 && S_ISREG(df->st_mode))
          {
            apr_file_printf(opts->out, "Running: %s %s\n", agent_file, data_file);
            *mod_pid = run_module(opts, agent_file, data_file);
            pid_string = apr_itoa(subpool, *mod_pid);
            apr_hash_set(pid_map, pid_string, APR_HASH_KEY_STRING, agent_file);
          }
          else
          {
            if (opts->verbose) apr_file_printf(opts->out, "Skipping (not a file): %s/%s\n", opts->agent_dir, file->d_name);
          }
        }
      }
      else
      {
        closedir(dir_h);
        break;
      }
    }

    while ((*cpid = wait(cstatus)) > 0)
    {
      cpid_string = apr_itoa(subpool, *cpid);
      module_name = apr_hash_get(pid_map, cpid_string, APR_HASH_KEY_STRING);
      apr_file_printf(opts->out, "Child %s (%s) terminated with ret code %i\n", module_name, cpid_string, *cstatus);
    }

    apr_file_printf(opts->out, "Loop end, sleeping for %i\n", opts->sleep_time);

    if (opts->run_once) break;

    apr_pool_destroy(subpool);
    sleep(opts->sleep_time);
  }
}

int run_module(fcm_opts_t *opts, char *agent_file, char *data_file)
{
  pid_t pID = fork();

  if (pID == 0)
  {
    if (opts->verbose) apr_file_printf(opts->err, "Child to exec '%s %s'\n", agent_file, data_file);
    execl(agent_file, agent_file, data_file, (char *) 0);
    apr_file_printf(opts->err, "Failed to exec '%s %s'\n", agent_file, data_file);
    return 0;
  }
  else if (pID < 0)
  {
    apr_file_printf(opts->err, "Failed to fork '%s %s'\n", agent_file, data_file);
    return 0;
  }
  return pID;
}
