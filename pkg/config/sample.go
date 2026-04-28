package config

// SampleYAML returns a ready-to-use configuration template.
func SampleYAML() string {
	return `kubeconfig: ""
output_dir: "./out"
file_prefix: "groot-capture"

collection:
  timeout: 20m
  worker_concurrency: 6
  namespaces:
    - kube-system
    - default
  targets:
    default:
      deployments:
        - api
      statefulsets:
        - redis
      daemonsets:
        - node-agent
      helm_releases:
        - my-release
  include_pod_logs: true
  include_previous_logs: true
  pod_log_tail_lines: 1500
  include_node_details: true
  extra_kubectl:
    - "get componentstatuses"
    - "get csr"

notify:
  slack:
    enabled: false
    webhook_url: ""

  teams:
    enabled: false
    webhook_url: ""

  telegram:
    enabled: false
    token: ""
    chat_id: ""
`
}
