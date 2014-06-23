#metric checks

## Usage

### Config File

The config file specifies the checks that will be done on the metrics values. An example:
```
[metric1]
check1 = metric1.Value < 16384
check2 = metric1.Value == 16384

[test rates]
check rate = metric1.Rate > 700
check rate 2 = metric2.Rate > 900

[check user percentage]
user 50 pct = cpustat.cpu.User.Value / cpustat.cpu.Total.Value * 100 > 50
user 30 pct = cpustat.cpu.User.Value / cpustat.cpu.Total.Value * 100 > 30
```
Each section title serves to describe its set of metric checks. Only the sections `default` and `nagios` are reserved for special information. The fields in the section will specify the checks. On the left hand side of the `=` is the name of the check; each name must be unique within its section. On the right hand side is the check that will be performed. Metrics can be specified by their full name, and will be replaced by the appropriate value.
