# Korrel8r Browser Demo

## Demo Objectives

- Present korrel8r as a tool for discovering and navigating related data in a cluster.
- Show some scenarios of realistic use for debugging alerts.
- Explain how open-ended it is, solicit ideas on additional applications.

## Setup

1. Log into an openshift cluster with openshift logging and Loki log store installed (Operator Hub)

1. Generate bad deployments to create alerts. **Note**: it should take at most 60s for alerts to become pending.
    ```
	oc apply -f bad-config-deployment.yaml -f bad-image-deployment.yaml
	```

1. Start korrel8r
    ```
	korrel8r web
	```

1. With Firefox browser:
   - Install side-view add-on https://addons.mozilla.org/en-CA/firefox/addon/side-view/
   - Open in side view: http://localhost:8080
   - Open in main window: openshift console

1. Create keyboard shortcut F1 to call script `./ytdkey` (e.g. using Gnome/Settings/Keyboard)
   - Copies URL from Firefox URL bar to the Korrel8r `start` field in side panel \
     **Note**:Assumes korrel8r is running in left-hand side-view panel, may be fragile.

The basic procedure for the demo is:-
 Navigate to an interesting object in the openshift console.
- Press F1 to copy the console URL to the korrel8r `start` field and click the `Update graph` button in the korrel8r UI.
- Click in korrel8r side-view to open openshift links in the main window.

This creates the impression that we are using korrel8r to navigate within the console.

## Script

### Introduction - show github home page

Korrel8r is an open-source tool for visualizing and navigating between related data in a cluster.
Project URL: https://github.com/korrel8r/korrel8r

- Currently in experimental stage but demonstrates the potential.

### First Scenario - Goal driven search

Looking at an alert, the operator wants to find related infrastructure logs.
- Show openshift console home screen.
- Find `KubeDeploymentReplicasMismatch - bad config`
- Open korrel8r in firefox side-view, note this is an experimental UI only.
- Choose Logs(infra) - goal-driven search
- Korrel8r displays path to logs. Click logs node.
- Logs open in openshift, filter with select/error: root cause is obvious - config file.
- Can click on other related objects.

Show alterantive goals 
- Select Events instead of logs, show in console
- Select Metrics, show in console.

Explain: 
- Possible to navigate alert - deployment - pod - logs with existing console links BUT
- Korrel8r is faster and gives a graphical view of related resources.

### Second Scenario - Neighbourhood

What if I don't know what to look for?
Neighbourhood shows all related objects:
- Korrel8r select "neighbourhood" and update.
- Note: Graph shows events, logs, deployment, pod and metrics - all connected to the Alert.
- Click several nodes to show we can jump to any of them in the console.

### How it works

Extendable set of:
- Domains: k8s resources, prometheus alerts and metrics, openshift-logging logs, more to come.
- Rules define relationships: within and between domains.

Korrel8r has a collection of rules relating different types of objects.
The rules form a static graph of all possible paths between related types of object.
- Korrel8r select "Rules", update
- Note: this is the set of all objects that could be reached from an alert and the rules that connect them.

Next korrel8r walks the graph.
Rules generate *queries* for various data stores, the queries retrieve live object data.
- Deselect "Rules" to go back to graph of live data.

### Wrap up

- Rules express relationships, open ended set of data types: k8s objects, metrics, logs, alerts ...
- Need to build a richer rule base, capture SRE debugging know-how.
- Need UI expertise to provide better presentations, integrated with consoles: sidebar? quick links? menus?
# Additional live talking points

Hover information (logs?)

API http://korrel8r-korrel8r.apps-crc.testing/api

curl -X GET http://korrel8r-korrel8r.apps-crc.testing/api/v1alpha1/domains

curl -X POST http://korrel8r-korrel8r.apps-crc.testing/api/v1alpha1/graphs/neighbours -d '{ "depth": 9, "start":{"class":"Deployment.apps.k8s","query": ["{\"namespace\":\"default\",\"kind\":\"Deployment\",\"group\":\"apps\",\"version\":\"v1\"}"]}}' | jq

Domains, Classes, Queries: Examples - refer to CURL query

Rules: Start->Goal, template, refer to rules yaml
  - Starting points are objects or queries.
  - Results are queries (could be objects)


