Configuration
------
```
# storage for parsed dmarc reports. Only SQL databases
# are supported.
storage:
  driver: sql
  params:
    connstr: 'sqlite:///tmp/dmarc-reports.sqlite'
# source for dmarc reports. supported drivers include
# imap/directory
# Reports fetched by IMAP are assumed to be in zip
# format as per DMARC specification
source:
  driver: imap
  params:
    username: xxxx
    password: xxxx
    ssl: true
    port: 993
    server: imap.gmail.com
   #folder:
# allowed IPs to send email per domain
# list one line per subnet
spf_authorized_ips:
  'examplecom':
    - 127.0.0.1/24
```

Storing aggregate reports
-------

```
bundle exec source/bin/dmarc-aggregate-recieve --config ~/dmarc-aggregate.yaml 
```

REST API
-------
```
bundle exec source/bin/dmarc-reports-rest-api
```

Examples
-------

Retrieving all reports (limit 100)
```
curl "http://localhost:8081//api/v1/reports" 
```

Retrieving first 10 reports
```
curl "http://localhost:8081//api/v1/reports/10"
```

Retrieve all records for domain "example.com"
```
curl "http://localhost:8081//api/v1/record/header_from/example.com"
```





