{{- /* Usage: {{ include "helm_lib_envs_for_proxy" . }} */ -}}
{{- define "helm_lib_envs_for_proxy" }}
  {{- $context := . -}}
  {{- if $context.Values.global.clusterConfiguration.proxy }}
    {{- if $context.Values.global.clusterConfiguration.proxy.httpProxy }}
- name: HTTP_PROXY
  value: {{ $context.Values.global.clusterConfiguration.proxy.httpProxy | quote }}
- name: http_proxy
  value: {{ $context.Values.global.clusterConfiguration.proxy.httpProxy | quote }}
    {{- end }}
    {{- if $context.Values.global.clusterConfiguration.proxy.httpsProxy }}
- name: HTTPS_PROXY
  value: {{ $context.Values.global.clusterConfiguration.proxy.httpsProxy | quote }}
- name: https_proxy
  value: {{ $context.Values.global.clusterConfiguration.proxy.httpsProxy | quote }}
    {{- end }}
    {{- $noProxy := list "127.0.0.1" "169.254.169.254" $context.Values.global.clusterConfiguration.clusterDomain $context.Values.global.clusterConfiguration.podSubnetCIDR $context.Values.global.clusterConfiguration.serviceSubnetCIDR }}
    {{- if $context.Values.global.clusterConfiguration.proxy.noProxy }}
      {{- $noProxy = concat $noProxy $context.Values.global.clusterConfiguration.proxy.noProxy }}
    {{- end }}
- name: NO_PROXY
  value: {{ $noProxy | join "," | quote }}
- name: no_proxy
  value: {{ $noProxy | join "," | quote }}
  {{- end }}
{{- end }}
