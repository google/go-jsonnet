{
  bar: {
    prometheusOperator+: {
      service+: {
        spec+: {
          ports: [
            {
              name: 'https',
              port: 8443,
              targetPort: 'https',
            },
          ],
        },
      },
      serviceMonitor+: {
        spec+: {
          endpoints: [
            {
              port: 'https',
              scheme: 'https',
              honorLabels: true,
              bearerTokenFile: '/var/run/secrets/kubernetes.io/serviceaccount/token',
              tlsConfig: {
                insecureSkipVerify: true,
              },
            },
          ],
        },
      },
      clusterRole+: {
        rules+: [
          {
            apiGroups: ['authentication.k8s.io'],
            resources: ['tokenreviews'],
            verbs: ['create'],
          },
          {
            apiGroups: ['authorization.k8s.io'],
            resources: ['subjectaccessreviews'],
            verbs: ['create'],
          },
        ],
      },
    },
    additional+: {
      '$schema': "http://json-schema.org/draft-07/schema#",
      '09': ['no', 'yes'],
    },
  },
  nothing: std.manifestYamlDoc(self.bar, indent_array_in_object=true, quote_keys=true),
}
