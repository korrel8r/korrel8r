local rule(host, service, port) = {
  host: host,
  http: {
    paths: [
      {
        backend: {
          service: {
            name: service,
            port: {
              number: port,
            },
          },
        },
        path: '/',
        pathType: 'Prefix',
      },
    ],
  },
};

{
  monitoring: {
    apiVersion: 'networking.k8s.io/v1',
    kind: 'Ingress',
    metadata: {
      name: 'web',
      namespace: 'monitoring',
    },
    spec: {
      rules: [
        rule('prometheus.127.0.0.1.nip.io', 'prometheus-k8s', 9090),
        rule('alertmanager.127.0.0.1.nip.io', 'alertmanager-main', 9093),
        rule('grafana.127.0.0.1.nip.io', 'grafana', 3000),
      ],
    },
  },

  logging: {
    apiVersion: 'networking.k8s.io/v1',
    kind: 'Ingress',
    metadata: {
      name: 'logging-web',
      namespace: 'logging',
    },
    spec: {
      rules: [
        rule('loki.127.0.0.1.nip.io', 'lokistack-dev-query-frontend-http', 3100),
      ],
    },
  },

  dashboard: {
    apiVersion: 'networking.k8s.io/v1',
    kind: 'Ingress',
    metadata: {
      name: 'dashboard',
      namespace: 'kubernetes-dashboard',
    },
    spec: {
      rules: [
        rule('dashboard.127.0.0.1.nip.io', 'dashboard-kubernetes-dashboard', 8080),
      ],
    },
  },
}
