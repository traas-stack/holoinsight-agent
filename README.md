# HoloInsight Agent

![License](https://img.shields.io/badge/license-Apache--2.0-green.svg)
[![Github stars](https://img.shields.io/github/stars/traas-stack/holoinsight-agent?style=flat-square])](https://github.com/traas-stack/holoinsight-agent)
[![OpenIssue](https://img.shields.io/github/issues/traas-stack/holoinsight-agent)](https://github.com/traas-stack/holoinsight-agent/issues)

HoloInsight Agent is responsible for collecting observability data for [HoloInsight](https://github.com/traas-stack/holoinsight).

# Overview
The HoloInsight Agent enables you to do the following:
- Collect system-level metrics from VMs/Pods/Nodes.
- Collect logs from VMs/Pods and aggregate locally according to the rules received from server side.
- Collect JVM stat metrics for VMs and Pods.
- Embedded part of the data collection capabilities of [telegraf](https://github.com/influxdata/telegraf) with enhanced configuration dynamic delivery capability.

# Features
1. Dynamic configuration delivery capability
2. K8s nodes/pods system metrics using [cAdvisor](https://github.com/google/cadvisor)
3. Generates metrics from log files in Pods
4. Collect JVM performance counter in Pods (such as heap/GC, like jstat)
5. No data loss when restarting and upgrading agent

# Build
```bash
sh ./scripts/docker/build.sh
```

# Install
### Docker Image
See [holoinsight/agent](https://hub.docker.com/r/holoinsight/agent)

# Licensing
HoloInsight Agent is under [Apache License 2.0](/LICENSE).
