local defaults = {
  local defaults = self,
  namespace: error 'must provide namespace',
  name: 'promtail',
  // try to be keep it in sync with the loki version.
  image: 'docker.io/grafana/promtail:2.7.3',
};

function(params) {
  local pt = self,
  _config:: defaults + params,

  _metadata:: {
    namespace: pt._config.namespace,
    name: pt._config.name,
  },

  daemonset: {
    apiVersion: 'apps/v1',
    kind: 'DaemonSet',
    metadata: pt._metadata,
    spec: {
      selector: {
        matchLabels: {
          name: pt._config.name,
        },
      },
      template: {
        metadata: {
          labels: {
            name: pt._config.name,
          },
        },
        spec: {
          containers: [
            {
              args: [
                '-config.file=/etc/promtail/promtail.yaml',
              ],
              env: [
                {
                  name: 'HOSTNAME',
                  valueFrom: {
                    fieldRef: {
                      fieldPath: 'spec.nodeName',
                    },
                  },
                },
              ],
              image: $._config.image,
              name: 'promtail',
              volumeMounts: [
                {
                  mountPath: '/var/log',
                  name: 'logs',
                },
                {
                  mountPath: '/etc/promtail',
                  name: 'promtail-config',
                },
                {
                  mountPath: '/var/lib/docker/containers',
                  name: 'varlibdockercontainers',
                  readOnly: true,
                },
              ],
            },
          ],
          serviceAccount: 'promtail',
          volumes: [
            {
              hostPath: {
                path: '/var/log',
              },
              name: 'logs',
            },
            {
              hostPath: {
                path: '/var/lib/docker/containers',
              },
              name: 'varlibdockercontainers',
            },
            {
              configMap: {
                name: pt._config.name + '-config',
              },
              name: 'promtail-config',
            },
          ],
        },
      },
    },
  },

  configmap: {
    apiVersion: 'v1',
    kind: 'ConfigMap',
    metadata: pt._metadata {
      name: pt._config.name + '-config',
    },
    data: {
      'promtail.yaml': std.manifestYamlDoc(
        {
          clients: [
            {
              tenant_id: 'aTenant',
              url: 'http://lokistack-dev-distributor-http.' + pt._config.namespace + '.svc:3100/loki/api/v1/push',
            },
          ],
          positions: {
            filename: '/tmp/positions.yaml',
          },
          scrape_configs: [
            {
              job_name: 'pod-logs',
              kubernetes_sd_configs: [
                {
                  role: 'pod',
                },
              ],
              pipeline_stages: [
                {
                  docker: {},
                },
              ],
              relabel_configs: [
                {
                  source_labels: [
                    '__meta_kubernetes_pod_node_name',
                  ],
                  target_label: '__host__',
                },
                {
                  action: 'labelmap',
                  regex: '__meta_kubernetes_pod_label_(.+)',
                },
                {
                  action: 'replace',
                  replacement: '$1',
                  separator: '/',
                  source_labels: [
                    '__meta_kubernetes_namespace',
                    '__meta_kubernetes_pod_name',
                  ],
                  target_label: 'job',
                },
                {
                  action: 'replace',
                  source_labels: [
                    '__meta_kubernetes_namespace',
                  ],
                  target_label: 'namespace',
                },
                {
                  action: 'replace',
                  source_labels: [
                    '__meta_kubernetes_pod_name',
                  ],
                  target_label: 'pod',
                },
                {
                  action: 'replace',
                  source_labels: [
                    '__meta_kubernetes_pod_container_name',
                  ],
                  target_label: 'container',
                },
                {
                  replacement: '/var/log/pods/*$1/*.log',
                  separator: '/',
                  source_labels: [
                    '__meta_kubernetes_pod_uid',
                    '__meta_kubernetes_pod_container_name',
                  ],
                  target_label: '__path__',
                },
              ],
            },
          ],
          server: {
            grpc_listen_port: 0,
            http_listen_port: 9080,
          },
          target_config: {
            sync_period: '10s',
          },
        }
      ),
    },
  },

  clusterrole: {
    apiVersion: 'rbac.authorization.k8s.io/v1',
    kind: 'ClusterRole',
    metadata: {
      name: pt._config.name + '-clusterrole',
    },
    rules: [
      {
        apiGroups: [
          '',
        ],
        resources: [
          'nodes',
          'services',
          'pods',
        ],
        verbs: [
          'get',
          'watch',
          'list',
        ],
      },
    ],
  },

  serviceaccount: {
    apiVersion: 'v1',
    kind: 'ServiceAccount',
    metadata: pt._metadata,
  },

  clusterrolebinding: {
    apiVersion: 'rbac.authorization.k8s.io/v1',
    kind: 'ClusterRoleBinding',
    metadata: {
      name: pt._config.name + '-clusterrolebinding',
    },
    roleRef: {
      apiGroup: 'rbac.authorization.k8s.io',
      kind: 'ClusterRole',
      name: pt._config.name + '-clusterrole',
    },
    subjects: [
      {
        kind: 'ServiceAccount',
        name: pt._config.name,
        namespace: pt._config.namespace,
      },
    ],
  },
}
