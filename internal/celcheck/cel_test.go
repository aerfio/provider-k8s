package celcheck_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"aerf.io/provider-k8s/internal/celcheck"
)

// tests based on example input from https://playcel.undistro.io/ + my own
func TestEval(t *testing.T) {
	tests := []struct {
		name       string
		expression string
		input      string
		want       bool
		wantErr    bool
	}{
		{
			name: "isUrl example",
			expression: `isURL(object.href) 
&& url(object.href).getScheme() == 'https' 
&& url(object.href).getHost() == 'example.com:80'
&& url(object.href).getHostname() == 'example.com'
&& url(object.href).getPort() == '80'
&& url(object.href).getEscapedPath() == '/path'
&& url(object.href).getQuery().size() == 1
`,
			input: `{
  "object": {
    "href": "https://user:pass@example.com:80/path?query=val#fragment"
  }
}
`,
			want: true,
		},
		{
			name: "jwt example",
			expression: `jwt.extra_claims.exists(c, c.startsWith('group'))
&& jwt.extra_claims
  .filter(c, c.startsWith('group'))
      .all(c, jwt.extra_claims[c]
          .all(g, g.endsWith('@acme.co')))`,
			input: `jwt: {
  "iss": "auth.acme.com:12350",
  "sub": "serviceAccount:delegate@acme.co",
  "aud": "my-project",
  "extra_claims": {
    "group1": [
      "admin@acme.co",
      "analyst@acme.co"
    ],
    "groupN": [
      "forever@acme.co"
    ],
    "labels": [ "metadata", "prod", "pii" ]
  }
}
`,
			want: true,
		},
		{
			name: "isQuantity example",
			expression: `isQuantity(object.memory) && 
quantity(object.memory)
  .add(quantity("700M"))
  .sub(1) // test without this subtraction
  .isLessThan(quantity(object.limit))`,
			input: `object:
  memory: 1.3G
  limit: 2G
`,
			want: true,
		},
		{
			name: "ready deployment",
			// https://github.com/kubernetes/kubernetes/blob/f927d5b385c9d6f8870cf9ae6c38cb96d54c23df/pkg/controller/deployment/util/deployment_util.go#L708
			expression: `has(metadata.generation) && 
has(status.observedGeneration) && 
metadata.generation>=status.observedGeneration &&
has(status.replicas) &&
has(status.updatedReplicas) &&
has(status.availableReplicas) &&
status.updatedReplicas == spec.replicas &&
status.replicas == spec.replicas && 
status.availableReplicas == spec.replicas`,
			input: readyDeploy,
			want:  true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := make(map[string]any)
			require.NoError(t, yaml.Unmarshal([]byte(tt.input), &input))

			got, err := celcheck.Eval(tt.expression, input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Eval() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Eval() got = %v, want %v", got, tt.want)
			}
		})
	}
}

const readyDeploy = `apiVersion: apps/v1
kind: Deployment
metadata:
  annotations:
    deployment.kubernetes.io/revision: "3"
  creationTimestamp: "2023-11-03T14:13:15Z"
  generation: 3
  labels:
    app.kubernetes.io/component: something
  name: deploy-name
  namespace: default
  resourceVersion: "40474632"
  uid: 6a6ba216-2689-42c5-ab3c-709b06b484f9
spec:
  progressDeadlineSeconds: 600
  replicas: 1
  revisionHistoryLimit: 10
  selector:
    matchLabels:
      app.kubernetes.io/component: something
  strategy:
    rollingUpdate:
      maxSurge: 25%
      maxUnavailable: 25%
    type: RollingUpdate
  template:
    metadata:
      annotations:
        kubectl.kubernetes.io/default-container: container
      creationTimestamp: null
      labels:
        app.kubernetes.io/component: some-component
    spec:
      containers:
      - image: image
        imagePullPolicy: IfNotPresent
        name: container-name
        resources:
          limits:
            cpu: 850m
            memory: 128Mi
          requests:
            cpu: 850m
            memory: 128Mi
        securityContext:
          allowPrivilegeEscalation: false
          capabilities:
            drop:
            - ALL
          privileged: false
          readOnlyRootFilesystem: true
          runAsNonRoot: true
      restartPolicy: Always
      schedulerName: default-scheduler
      securityContext:
        runAsNonRoot: true
      serviceAccount: example-sa
      serviceAccountName: example-sa
      terminationGracePeriodSeconds: 30
status:
  availableReplicas: 1
  conditions:
  - lastTransitionTime: "2023-11-03T14:13:15Z"
    lastUpdateTime: "2023-11-07T09:08:17Z"
    message: ReplicaSet "some-replicaset" has successfully progressed.
    reason: NewReplicaSetAvailable
    status: "True"
    type: Progressing
  - lastTransitionTime: "2023-11-07T14:44:52Z"
    lastUpdateTime: "2023-11-07T14:44:52Z"
    message: Deployment has minimum availability.
    reason: MinimumReplicasAvailable
    status: "True"
    type: Available
  observedGeneration: 3
  readyReplicas: 1
  replicas: 1
  updatedReplicas: 1`
