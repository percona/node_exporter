# Percona Node exporter [![Build Status]

Prometheus exporter for machine metrics, written in Go with pluggable metric
collectors.

## Collectors

There is varying support for collectors on each operating system. The tables
below list all existing collectors and the supported systems.

Which collectors are used is controlled by the `--collectors.enabled` flag.

### Enabled by default

Name     | Description | OS
---------|-------------|----
conntrack | Shows conntrack statistics (does nothing if no `/proc/sys/net/netfilter/` present). | Linux
cpu | Exposes CPU statistics | Dragonfly, FreeBSD
diskstats | Exposes disk I/O statistics from `/proc/diskstats`. | Linux
entropy | Exposes available entropy. | Linux
filefd | Exposes file descriptor statistics from `/proc/sys/fs/file-nr`. | Linux
filesystem | Exposes filesystem statistics, such as disk space used. | FreeBSD, Dragonfly, Linux, OpenBSD
hwmon | Expose hardware monitoring and sensor data from `/sys/class/hwmon/`. | Linux
loadavg | Exposes load average. | Darwin, Dragonfly, FreeBSD, Linux, NetBSD, OpenBSD, Solaris
mdadm | Exposes statistics about devices in `/proc/mdstat` (does nothing if no `/proc/mdstat` present). | Linux
meminfo | Exposes memory statistics. | Dragonfly, FreeBSD, Linux
netdev | Exposes network interface statistics such as bytes transferred. | Dragonfly, FreeBSD, Linux, OpenBSD
netstat | Exposes network statistics from `/proc/net/netstat`. This is the same information as `netstat -s`. | Linux
stat | Exposes various statistics from `/proc/stat`. This includes CPU usage, boot time, forks and interrupts. | Linux
textfile | Exposes statistics read from local disk. The `--collector.textfile.directory` flag must be set. | _any_
time | Exposes the current system time. | _any_
vmstat | Exposes statistics from `/proc/vmstat`. | Linux


### Disabled by default

Name     | Description | OS
---------|-------------|----
bonding | Exposes the number of configured and active slaves of Linux bonding interfaces. | Linux
devstat | Exposes device statistics | Dragonfly, FreeBSD
gmond | Exposes statistics from Ganglia. | _any_
interrupts | Exposes detailed interrupts statistics. | Linux, OpenBSD
ipvs | Exposes IPVS status from `/proc/net/ip_vs` and stats from `/proc/net/ip_vs_stats`. | Linux
ksmd | Exposes kernel and system statistics from `/sys/kernel/mm/ksm`. | Linux
logind | Exposes session counts from [logind](http://www.freedesktop.org/wiki/Software/systemd/logind/). | Linux
megacli | Exposes RAID statistics from MegaCLI. | Linux
meminfo_numa | Exposes memory statistics from `/proc/meminfo_numa`. | Linux
ntp | Exposes time drift from an NTP server. | _any_
runit | Exposes service status from [runit](http://smarden.org/runit/). | _any_
supervisord | Exposes service status from [supervisord](http://supervisord.org/). | _any_
systemd | Exposes service and system status from [systemd](http://www.freedesktop.org/wiki/Software/systemd/). | Linux
tcpstat | Exposes TCP connection status information from `/proc/net/tcp` and `/proc/net/tcp6`. (Warning: the current version has potential performance issues in high load situations.) | Linux

### Textfile Collector

The textfile collector is similar to the [Pushgateway](https://github.com/prometheus/pushgateway),
in that it allows exporting of statistics from batch jobs. It can also be used
to export static metrics, such as what role a machine has. The Pushgateway
should be used for service-level metrics. The textfile module is for metrics
that are tied to a machine.

To use it, set the `--collector.textfile.directory` flag on the Node exporter. The
collector will parse all files in that directory matching the glob `*.prom`
using the [text
format](http://prometheus.io/docs/instrumenting/exposition_formats/).

To atomically push completion time for a cron job:
```
echo my_batch_job_completion_time $(date +%s) > /path/to/directory/my_batch_job.prom.$$
mv /path/to/directory/my_batch_job.prom.$$ /path/to/directory/my_batch_job.prom
```

To statically set roles for a machine using labels:
```
echo 'role{role="application_server"} 1' > /path/to/directory/role.prom.$$
mv /path/to/directory/role.prom.$$ /path/to/directory/role.prom
```

## Building and running

    make
    ./node_exporter <flags>

## Running tests

    make test


## Using Docker

You can deploy this exporter using the [prom/node-exporter](https://registry.hub.docker.com/u/prom/node-exporter/) Docker image.

For example:

```bash
docker pull prom/node-exporter

docker run -d -p 9100:9100 --net="host" prom/node-exporter
```


[travis]: https://travis-ci.org/prometheus/node_exporter
[hub]: https://hub.docker.com/r/prom/node-exporter/
[circleci]: https://circleci.com/gh/prometheus/node_exporter
[quay]: https://quay.io/repository/prometheus/node-exporter


## Visualize

There is a Grafana dashboard for Host available as a part of [PMM](https://www.percona.com/doc/percona-monitoring-and-management/index.html) project, you can see the demo [here](https://pmmdemo.percona.com/graph/dashboard/db/system-overview).

## Submit Bug Report
If you find a bug in Percona Node Exporter or one of the related projects, you should submit a report to that project's [JIRA](https://jira.percona.com) issue tracker.

Your first step should be  [to search](https://jira.percona.com/issues/?jql=project=PMM%20AND%20component=Node_Exporter)  the existing set of open tickets for a similar report. If you find that someone else has already reported your problem, then you can upvote that report to increase its visibility.

If there is no existing report, submit a report following these steps:

1.  [Sign in to Percona JIRA.](https://jira.percona.com/login.jsp) You will need to create an account if you do not have one.
2.  [Go to the Create Issue screen and select the relevant project.](https://jira.percona.com/secure/CreateIssueDetails!init.jspa?pid=11600&issuetype=1&priority=3&components=11710)
3.  Fill in the fields of Summary, Description, Steps To Reproduce, and Affects Version to the best you can. If the bug corresponds to a crash, attach the stack trace from the logs.

An excellent resource is [Elika Etemad's article on filing good bug reports.](http://fantasai.inkedblade.net/style/talks/filing-good-bugs/).

As a general rule of thumb, please try to create bug reports that are:

-   *Reproducible.* Include steps to reproduce the problem.
-   *Specific.* Include as much detail as possible: which version, what environment, etc.
-   *Unique.* Do not duplicate existing tickets.
-   *Scoped to a Single Bug.* One bug per report.

