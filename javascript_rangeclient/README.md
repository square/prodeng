rangeclient synopsis
============================
  * include jquery, range.js

  * Create range object for communication with ranged

```
  r = new Range("range","80");
```

Examples
-------

use ranged to expand the range expression into an Array

"foo10..12" => [ foo10, foo11, foo12 ] OR %foo => [ foo10, foo11, foo12 ]

```
hosts =  r.expand(rangearg)
```

use ranged to compress the array of hostnames into a range expression

[ foo10, foo11, foo12 ] => "foo10..12"
  
```
range_exp = r.compress(hosts)
```
