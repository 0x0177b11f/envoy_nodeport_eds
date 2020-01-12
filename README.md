
# Envoy k8s nodeport service eds service

## Command line options

```bash
  -addressType string
        node address type (default "InternalIP")
  -intervals int
        update intervals (default 30)
  -kubeconfig string
        absolute path to the kubeconfig file (default "${HOME}/.kube/config")
  -listener string
        listener address (default "0.0.0.0")
  -namespace string
        kube namespace (default "default")
  -port uint
        listener port (default 8000)
  -serviceName string
        service name (default is all nodeport service) (default "*")
```

## k8s ServiceAccount Sample

[rbac.yaml](./rbac.yaml)

## Envoy Config Sample

```yaml
node:
  cluster: "lb_server"
  id: "default" # <- k8s namespace

admin:
  access_log_path: /tmp/admin_access.log
  address:
    socket_address:
      protocol: TCP
      address: 127.0.0.1
      port_value: 9901

static_resources:
  listeners:
  - name: http_proxy
    address:
      socket_address:
        protocol: TCP
        address: 0.0.0.0
        port_value: 80
    filter_chains:
    - filters:
      - name: envoy.tcp_proxy
        typed_config:
          "@type": type.googleapis.com/envoy.config.filter.network.tcp_proxy.v2.TcpProxy
          stat_prefix: ingress_tcp
          cluster: web
          access_log:
            - name: envoy.file_access_log
              typed_config:
                "@type": type.googleapis.com/envoy.config.accesslog.v2.FileAccessLog
                path: /dev/stdout
  - name: https_proxy
    address:
      socket_address:
        protocol: TCP
        address: 0.0.0.0
        port_value: 443
    filter_chains:
    - filters:
      - name: envoy.tcp_proxy
        typed_config:
          "@type": type.googleapis.com/envoy.config.filter.network.tcp_proxy.v2.TcpProxy
          stat_prefix: ingress_tcp
          cluster: websecure
          access_log:
            - name: envoy.file_access_log
              typed_config:
                "@type": type.googleapis.com/envoy.config.accesslog.v2.FileAccessLog
                path: /dev/stdout
  clusters:
  - name: web  # <- this is k8s nodeport service port name
    connect_timeout: 30s
    lb_policy: ROUND_ROBIN
    type: EDS
    eds_cluster_config:
      eds_config:
        api_config_source:
          api_type: GRPC
          grpc_services:
            envoy_grpc:
              cluster_name: xds_cluster
  - name: websecure  # <- this is k8s nodeport service port name
    connect_timeout: 30s
    lb_policy: ROUND_ROBIN
    type: EDS
    eds_cluster_config:
      eds_config:
        api_config_source:
          api_type: GRPC
          grpc_services:
            envoy_grpc:
              cluster_name: xds_cluster
  - name: xds_cluster
    connect_timeout: 1s
    type: STATIC
    lb_policy: ROUND_ROBIN
    http2_protocol_options: {}
    upstream_connection_options:
      tcp_keepalive: {}
    load_assignment:
      cluster_name: xds_cluster
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address:
                address: 127.0.0.1 # <- envoy_nodeport_eds address
                port_value: 8000 # <- envoy_nodeport_eds port
```
