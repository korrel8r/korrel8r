local kp =
  (import 'kube-prometheus/main.libsonnet') +
  // Uncomment the following imports to enable its patches
  // (import 'kube-prometheus/addons/anti-affinity.libsonnet') +
  // (import 'kube-prometheus/addons/managed-cluster.libsonnet') +
  // (import 'kube-prometheus/addons/node-ports.libsonnet') +
  // (import 'kube-prometheus/addons/static-etcd.libsonnet') +
  // (import 'kube-prometheus/addons/custom-metrics.libsonnet') +
  // (import 'kube-prometheus/addons/external-metrics.libsonnet') +
  // (import 'kube-prometheus/addons/pyrra.libsonnet') +
  {
    values+:: {
      common+: {
        namespace: 'monitoring',
      },
      alertmanager+: {
        replicas: 1,
      },
      prometheus+: {
        replicas: 1,
      },
      prometheusAdapter+: {
        replicas: 1,
      },
      grafana+: {
        datasources: [
          {
            name: 'prometheus',
            type: 'prometheus',
            access: 'proxy',
            orgId: 1,
            url: 'http://prometheus-k8s.monitoring.svc:9090',
          },
          {
            name: 'loki',
            type: 'loki',
            access: 'proxy',
            orgId: 1,
            url: 'http://lokistack-dev-query-frontend-http.logging.svc:3100',
            jsonData: {
              httpHeaderName1: 'X-Scope-OrgID',
            },
            secureJsonData: {
              httpHeaderValue1: 'aTenant',
            },
          },
        ],
      },
    },
  } + {
    prometheus+: {
      tokenSecret: {
        apiVersion: 'v1',
        kind: 'Secret',
        type: 'kubernetes.io/service-account-token',
        metadata: {
          name: 'prometheus-k8s-token',
          namespace: 'monitoring',
          annotations: {
            'kubernetes.io/service-account.name': 'prometheus-k8s',
          },
        },
      },

      prometheus+: {
        spec+: {
          retentionSize: '1GiB',
          storage: {
            volumeClaimTemplate: {
              spec: {
                resources: {
                  requests: {
                    storage: '1Gi',
                  },
                },
              },
            },
          },
        },
      },
    },
  } + {
    kubernetesControlPlane+: {
      serviceMonitorKubeScheduler: null,
      serviceMonitorKubeControllerManager: null,
      podMonitorKubeScheduler: {
        apiVersion: 'monitoring.coreos.com/v1',
        kind: 'PodMonitor',
        metadata: {
          name: 'kube-scheduler',
          namespace: 'monitoring',
          labels: { component: 'kube-scheduler' },
        },
        spec: {
          jobLabel: 'component',
          podMetricsEndpoints: [{
            port: 'https',
            interval: '30s',
            scheme: 'https',
            bearerTokenSecret: {
              name: 'prometheus-k8s-token',
              key: 'token',
            },
            tlsConfig: { insecureSkipVerify: true },
          }],
          selector: {
            matchLabels: { component: 'kube-scheduler' },
          },
          namespaceSelector: {
            matchNames: ['kube-system'],
          },
        },
      },
      podMonitorKubeControllerManager: {
        apiVersion: 'monitoring.coreos.com/v1',
        kind: 'PodMonitor',
        metadata: {
          name: 'kube-controller-manager',
          namespace: 'monitoring',
          labels: { component: 'kube-controller-manager' },
        },
        spec: {
          jobLabel: 'component',
          podMetricsEndpoints: [{
            port: 'https',
            interval: '30s',
            scheme: 'https',
            bearerTokenSecret: {
              name: 'prometheus-k8s-token',
              key: 'token',
            },
            tlsConfig: { insecureSkipVerify: true },
          }],
          selector: {
            matchLabels: { component: 'kube-controller-manager' },
          },
          namespaceSelector: {
            matchNames: ['kube-system'],
          },
        },
      },
    },
  };

local ingress = (import 'ingress.libsonnet');

local promtail = (import 'promtail.libsonnet');
local pt = promtail({ namespace: 'logging' });


{ 'setup/0namespace-namespace': kp.kubePrometheus.namespace } +
{
  ['setup/prometheus-operator-' + name]: kp.prometheusOperator[name]
  for name in std.filter((function(name) name != 'serviceMonitor' && name != 'prometheusRule'), std.objectFields(kp.prometheusOperator))
} +
// { 'setup/pyrra-slo-CustomResourceDefinition': kp.pyrra.crd } +
// serviceMonitor and prometheusRule are separated so that they can be created after the CRDs are ready
{ 'prometheus-operator-serviceMonitor': kp.prometheusOperator.serviceMonitor } +
{ 'prometheus-operator-prometheusRule': kp.prometheusOperator.prometheusRule } +
{ 'kube-prometheus-prometheusRule': kp.kubePrometheus.prometheusRule } +
{ ['alertmanager-' + name]: kp.alertmanager[name] for name in std.objectFields(kp.alertmanager) } +
{ ['blackbox-exporter-' + name]: kp.blackboxExporter[name] for name in std.objectFields(kp.blackboxExporter) } +
{ ['grafana-' + name]: kp.grafana[name] for name in std.objectFields(kp.grafana) } +
// { ['pyrra-' + name]: kp.pyrra[name] for name in std.objectFields(kp.pyrra) if name != 'crd' } +
{ ['kube-state-metrics-' + name]: kp.kubeStateMetrics[name] for name in std.objectFields(kp.kubeStateMetrics) } +
{ ['kubernetes-' + name]: kp.kubernetesControlPlane[name] for name in std.objectFields(std.prune(kp.kubernetesControlPlane)) }
{ ['node-exporter-' + name]: kp.nodeExporter[name] for name in std.objectFields(kp.nodeExporter) } +
{ ['prometheus-' + name]: kp.prometheus[name] for name in std.objectFields(kp.prometheus) } +
{ ['prometheus-adapter-' + name]: kp.prometheusAdapter[name] for name in std.objectFields(kp.prometheusAdapter) } +
{ ['promtail-' + name]: pt[name] for name in std.objectFields(pt) } +
{ ['ingress-' + name]: ingress[name] for name in std.objectFields(ingress) } +
{
  'lokistack-dev': {
    apiVersion: 'loki.grafana.com/v1',
    kind: 'LokiStack',
    metadata: {
      name: 'lokistack-dev',
      namespace: 'logging',
    },
    spec: {
      size: '1x.extra-small',
      storage: {
        schemas: [
          { effectiveDate: '2022-06-01', version: 'v12' },
        ],
        secret: { name: 'test', type: 's3' },
      },
      storageClassName: 'standard',
    },
  },
}
