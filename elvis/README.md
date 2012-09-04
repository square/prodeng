Elvis synopsis
============================

  system "id"
  Elvis.run_as(nobody) {
    system "id"
  }
  system "id"


Elvis limitations
============================

 * Elvis currently supports linux and OSX. set*id() functions behave very differently on different unixes, patches welcome.
 * Elvis cannot be used with JRuby because of the JVM model. Instead, spawn a subprocess.
 * Elvis cannot be used in conjunction with threads, as it modifies process global state.
