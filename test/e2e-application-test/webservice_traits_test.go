/*
Copyright 2026 The KubeVela Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package application_test

import (
	"context"

	"github.com/guidewire-oss/sawchain"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// traitCase describes one webservice Application with traits applied.
// The case name doubles as the Application, component, and Deployment name.
type traitCase struct {
	name string
	// traits is a YAML list injected into the component via (parse_yaml($traits)).
	traits string
	// expected is a multi-document Chainsaw template of resources that must exist.
	// Each Deployment document writes its own header.
	expected string
	// absent is an optional multi-document template; each document must match nothing.
	absent string
}

const appHealthy = `
	apiVersion: core.oam.dev/v1beta1
	kind: Application
	metadata:
	  name: ($name)
	  namespace: ($namespace)
	status:
	  status: running
	  services:
	  - name: ($name)
	    healthy: true
`

var traitCases = []traitCase{
	{
		name: "scaler",
		traits: `
			- type: scaler
			  properties:
			    replicas: 2
		`,
		expected: `
			apiVersion: apps/v1
			kind: Deployment
			metadata:
			  name: ($name)
			  namespace: ($namespace)
			spec:
			  replicas: 2
		`,
	},
	{
		name: "labels",
		traits: `
			- type: labels
			  properties:
			    tier: web
		`,
		expected: `
			apiVersion: apps/v1
			kind: Deployment
			metadata:
			  name: ($name)
			  namespace: ($namespace)
			  labels:
			    tier: web
			spec:
			  template:
			    metadata:
			      labels:
			        tier: web
		`,
	},
	{
		name: "annotations",
		traits: `
			- type: annotations
			  properties:
			    team: platform
		`,
		expected: `
			apiVersion: apps/v1
			kind: Deployment
			metadata:
			  name: ($name)
			  namespace: ($namespace)
			  annotations:
			    team: platform
			spec:
			  template:
			    metadata:
			      annotations:
			        team: platform
		`,
	},
	{
		name: "env",
		traits: `
			- type: env
			  properties:
			    env:
			      FOO: bar
		`,
		expected: `
			apiVersion: apps/v1
			kind: Deployment
			metadata:
			  name: ($name)
			  namespace: ($namespace)
			spec:
			  template:
			    spec:
			      containers:
			      - name: ($name)
			        env:
			        - name: FOO
			          value: bar
		`,
	},
	{
		name: "resource",
		traits: `
			- type: resource
			  properties:
			    cpu: 100m
			    memory: 64Mi
		`,
		expected: `
			apiVersion: apps/v1
			kind: Deployment
			metadata:
			  name: ($name)
			  namespace: ($namespace)
			spec:
			  template:
			    spec:
			      containers:
			      - name: ($name)
			        resources:
			          requests:
			            cpu: 100m
			            memory: 64Mi
			          limits:
			            cpu: 100m
			            memory: 64Mi
		`,
	},
	{
		name: "sidecar",
		traits: `
			- type: sidecar
			  properties:
			    name: helper
			    image: busybox:1.34
			    cmd: ["sleep", "86400"]
		`,
		expected: `
			apiVersion: apps/v1
			kind: Deployment
			metadata:
			  name: ($name)
			  namespace: ($namespace)
			spec:
			  template:
			    spec:
			      containers:
			      - name: ($name)
			      - name: helper
			        image: busybox:1.34
			        command: ["sleep", "86400"]
		`,
	},
	{
		name: "lifecycle",
		traits: `
			- type: lifecycle
			  properties:
			    preStop:
			      exec:
			        command: ["sleep", "1"]
		`,
		expected: `
			apiVersion: apps/v1
			kind: Deployment
			metadata:
			  name: ($name)
			  namespace: ($namespace)
			spec:
			  template:
			    spec:
			      containers:
			      - name: ($name)
			        lifecycle:
			          preStop:
			            exec:
			              command: ["sleep", "1"]
		`,
	},
	{
		name: "container-ports",
		traits: `
			- type: container-ports
			  properties:
			    ports:
			    - containerPort: 9090
		`,
		expected: `
			apiVersion: apps/v1
			kind: Deployment
			metadata:
			  name: ($name)
			  namespace: ($namespace)
			spec:
			  template:
			    spec:
			      containers:
			      - name: ($name)
			        ports:
			        - containerPort: 8080
			        - containerPort: 9090
			          protocol: TCP
		`,
	},
	{
		name: "hostalias",
		traits: `
			- type: hostalias
			  properties:
			    hostAliases:
			    - ip: 10.1.2.3
			      hostnames: [a.example.com]
		`,
		expected: `
			apiVersion: apps/v1
			kind: Deployment
			metadata:
			  name: ($name)
			  namespace: ($namespace)
			spec:
			  template:
			    spec:
			      hostAliases:
			      - ip: 10.1.2.3
			        hostnames: [a.example.com]
		`,
	},
	{
		name: "k8s-update-strategy",
		traits: `
			- type: k8s-update-strategy
			  properties:
			    strategy:
			      type: Recreate
		`,
		expected: `
			apiVersion: apps/v1
			kind: Deployment
			metadata:
			  name: ($name)
			  namespace: ($namespace)
			spec:
			  strategy:
			    type: Recreate
		`,
	},
	{
		// The labels trait runs first so json-patch has a label to remove.
		name: "json-patch",
		traits: `
			- type: labels
			  properties:
			    to-delete: "yes"
			- type: json-patch
			  properties:
			    operations:
			    - op: add
			      path: /spec/replicas
			      value: 3
			    - op: add
			      path: /spec/template/metadata/labels/patched
			      value: "yes"
			    - op: remove
			      path: /spec/template/metadata/labels/to-delete
			    - op: add
			      path: /spec/template/spec/containers/1
			      value:
			        name: helper
			        image: busybox:1.34
			        command: ["sleep", "86400"]
		`,
		expected: `
			apiVersion: apps/v1
			kind: Deployment
			metadata:
			  name: ($name)
			  namespace: ($namespace)
			spec:
			  replicas: 3
			  template:
			    metadata:
			      labels:
			        patched: "yes"
			    spec:
			      containers:
			      - name: ($name)
			      - name: helper
			        image: busybox:1.34
		`,
		absent: `
			apiVersion: apps/v1
			kind: Deployment
			metadata:
			  name: ($name)
			  namespace: ($namespace)
			spec:
			  template:
			    metadata:
			      labels:
			        to-delete: "yes"
		`,
	},
	{
		name: "json-merge-patch",
		traits: `
			- type: labels
			  properties:
			    to-delete: "yes"
			- type: json-merge-patch
			  properties:
			    metadata:
			      labels:
			        deploy-label: added
			    spec:
			      replicas: 3
			      template:
			        metadata:
			          labels:
			            to-delete: null
		`,
		expected: `
			apiVersion: apps/v1
			kind: Deployment
			metadata:
			  name: ($name)
			  namespace: ($namespace)
			  labels:
			    deploy-label: added
			spec:
			  replicas: 3
		`,
		absent: `
			apiVersion: apps/v1
			kind: Deployment
			metadata:
			  name: ($name)
			  namespace: ($namespace)
			spec:
			  template:
			    metadata:
			      labels:
			        to-delete: "yes"
		`,
	},
	{
		// Auxiliary resource: full Service spec, as in Sawchain's gateway example.
		name: "expose",
		traits: `
			- type: expose
			  properties:
			    ports:
			    - port: 8080
		`,
		expected: `
			apiVersion: v1
			kind: Service
			metadata:
			  name: ($name)
			  namespace: ($namespace)
			spec:
			  type: ClusterIP
			  selector:
			    app.oam.dev/component: ($name)
			  ports:
			  - name: port-8080
			    port: 8080
			    targetPort: 8080
			    protocol: TCP
		`,
	},
}

var _ = Describe("Webservice component traits (Sawchain)", Label("core-application"), func() {
	var (
		ctx       context.Context
		sc        *sawchain.Sawchain
		namespace string
	)

	BeforeEach(func() {
		ctx = context.Background()
		namespace = randomNamespaceName("webservice-traits")
		Expect(k8sClient.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}})).To(Succeed())
		sc = sawchain.New(GinkgoTB(), k8sClient, "2m", "2s", map[string]any{"namespace": namespace})
	})

	AfterEach(func() {
		Expect(k8sClient.Delete(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: namespace}})).To(Succeed())
	})

	for _, c := range traitCases {
		It("applies the "+c.name+" trait", func() {
			bindings := map[string]any{"name": c.name, "traits": c.traits}
			sc.CreateAndWait(ctx, "testdata/webservice/traits.tpl.yaml", bindings)

			Eventually(sc.CheckFunc(ctx, c.expected, bindings), sawchainTimeout, sawchainInterval).Should(Succeed())
			Eventually(sc.CheckFunc(ctx, appHealthy, bindings), sawchainTimeout, sawchainInterval).Should(Succeed())
			if c.absent != "" {
				Expect(sc.CheckFunc(ctx, c.absent, bindings)()).NotTo(Succeed())
			}
		})
	}
})
