---
name: Feature request
about: Suggest an idea for this project
title: "[RFE]"
labels: enhancement
assignees: ''

---

**1. When I am in this situation: ＿＿＿＿**

Situations where:
- you have some information, and want to use it to jump to related information
- you know how get there, but it’s not trivial: you have to click many console screens, type many commands, write scripts or other automated tools.

The context could be
- interacting with a cluster via graphical console or command line.
- building services that will run in a cluster to collect or analyze data.
- out-of-cluster analysis of cluster data.

**2. And I am looking at: ＿＿＿＿**

Any type of signal or cluster data: metrics, traces, logs alerts, k8s events, k8s resources, network events, add your own…

The data could be viewed on a console, printed by command line tools, available from files or stores (Loki, Prometheus …)

**3. I would like to see: ＿＿＿＿**

Again types of information include: metrics, traces, logs alerts, k8s events, k8s resources, network events, add your own…

Describe the desired data, and the steps needed to get from the starting point in step 2.

Examples:
- I’m looking at this alert, and I want to see …
- I’m looking at this k8s Event, and I want to see …
- There are reports of slow responses from this Service, I want to see…
- CPU/Memory is getting scarce on this node, I want to see…
- These PVs are filling up, I want to see…
- Cluster is using more storage than I expected, I want to see…
