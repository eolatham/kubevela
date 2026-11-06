# Sawchain e2e tests for webservice traits: plan and status

Location: `test/e2e-application-test/`

## Design (decided)
- One table, `traitCases` in `webservice_traits_test.go`. Each `traitCase` has `name`, `traits`, `expected` and an optional `absent`.
  - `name` is also the Application, component and Deployment name, so cases don't collide in one namespace.
  - `traits` is a YAML list passed as a binding. The template `testdata/webservice/traits.tpl.yaml` injects it with `traits: (parse_yaml($traits))`.
  - `expected` is multi-doc Chainsaw YAML checked with one `CheckFunc`. Each Deployment doc writes its own header. Auxiliary resources (Service, Ingress, HPA) get their full spec asserted, as in Sawchain's examples.
  - `absent` is checked once with `Expect(...()).NotTo(Succeed())`, after `expected` has passed. It covers removal ops (json-patch `remove`, merge-patch `null`).
- Every case also asserts Application health: `status.status: running` and `status.services[name].healthy: true`. The field is `status.status`, not `status.phase`.
- The scaler add/remove lifecycle stays in `webservice_test.go` as an Ordered test, since a one-shot table can't cover updates.
- `trait_test.go` (old json-patch and json-merge-patch tests) was replaced and deleted. Nothing else used `readAppFromFile`.

## Sawchain findings
- Bindings are injected as-is. A raw YAML string binding with `traits: ($traits)` becomes a string, so use `(parse_yaml($traits))`.
- Empty `$traits` (`""`, `"null"`, `"# none"`) renders `traits: null`. `"[]"` renders `traits: []`. Rendering only: not yet confirmed that the API server accepts `traits: null` on create. Fall back to `"[]"` if it doesn't.
- `CheckFunc` accepts multi-doc templates. It splits on `---` once, checks the documents in order on each poll, and reports the first mismatch.

## Done
Cases in the table: scaler, labels, annotations, env, resource, sidecar, lifecycle, container-ports, hostalias, k8s-update-strategy, json-patch, json-merge-patch, expose.
Status: compiles (`go vet`, `gofmt` clean). **Never run**: no cluster was available. Expected YAML was written from the `.cue` definitions in `vela-templates/definitions/internal/trait/`, so expect some fixes on the first real run.

## Next
1. **Run against a cluster with KubeVela and fix mismatches.** Check these first:
   - `traits: null` on create.
   - Trait order in the `json-patch` and `json-merge-patch` cases (`labels` runs before the patch).
   - `container-ports` default container name and port ordering.
   - The `resource` cpu and memory normalisation.
2. **Add remaining Tier 1 traits:**
   - `command` and `startup-probe`: `oamdev/testapp` may lack `sleep`, so choose commands that don't crash the pod.
   - `securitycontext`: set `runAsNonRoot: false` if the image runs as root.
   - Also `podsecuritycontext`, `affinity`, `topologyspreadconstraints`, `container-image`, `init-container`.
   - `service-account` and `storage` need an optional `fixtures` field (a ServiceAccount, a ConfigMap), applied with `CreateAndWait` before the Application.
3. **Tier 2:** `gateway`, `pure-ingress`, `hpa` and `cpuscaler`, each asserting the full Ingress or HPA spec. The HPA needs metrics-server, so check that first.
4. **Skip for now:** `nocalhost`, `service-binding` (external setup needed).
5. **Cleanup:** drop `expose: true` from the scaler lifecycle test if it should match the table base (the table base has no `expose`, because the `expose` trait creates a same-named Service). Run `/code-review` and `/simplify` before committing.
