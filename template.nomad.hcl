job "alveoli" {
  datacenters = ["dc1"]
  type        = "service"

  update {
    max_parallel     = 1
    min_healthy_time = "30s"
    healthy_deadline = "3m"
    auto_revert      = true
    canary           = 0
  }

  group "alveoli" {
    constraint {
      operator  = "distinct_hosts"
      value     = "true"
    }
    vault {
      policies      = ["nomad-tls-storer"]
      change_mode   = "signal"
      change_signal = "SIGUSR1"
      env           = false
    }

    count = 3

    restart {
      attempts = 10
      interval = "5m"
      delay    = "15s"
      mode     = "delay"
    }

    ephemeral_disk {
      size = 200
    }

    task "server" {
      driver = "docker"

      env {
        HTTPS_PROXY="http://http.proxy.discovery.fr-par.vx-labs.net:3128"
        CONSUL_HTTP_ADDR = "$${NOMAD_IP_http}:8500"
        VAULT_ADDR       = "http://active.vault.service.consul:8200/"
      }

      template {
        change_mode = "restart"
        destination = "local/environment"
        env         = true

        data = <<EOH
{{with secret "secret/data/vx/mqtt"}}
LE_EMAIL="{{.Data.acme_email}}"
ALVEOLI_AUTH0_CLIENT_DOMAIN="{{ .Data.auth0_client_domain }}"
ALVEOLI_AUTH0_API_ID="{{ .Data.auth0_api_id }}"
ALVEOLI_RPC_TLS_CERTIFICATE_FILE="{{ env "NOMAD_TASK_DIR" }}/cert.pem"
ALVEOLI_RPC_TLS_PRIVATE_KEY_FILE="{{ env "NOMAD_TASK_DIR" }}/key.pem"
ALVEOLI_RPC_TLS_CERTIFICATE_AUTHORITY_FILE="{{ env "NOMAD_TASK_DIR" }}/ca.pem"
ALVEOLI_SUBSCRIPTIONS_MQTT_BROKER_SNI="broker.iot.cloud.vx-labs.net"
ALVEOLI_SUBSCRIPTIONS_MQTT_BROKER="10.64.72.135"
no_proxy="10.0.0.0/8,172.16.0.0/12,*.service.consul"
{{end}}
        EOH
      }

      template {
        change_mode = "restart"
        destination = "local/cert.pem"
        splay       = "1h"

        data = <<EOH
{{- $cn := printf "common_name=%s" (env "NOMAD_ALLOC_ID") -}}
{{- $ipsans := printf "ip_sans=%s" (env "NOMAD_IP_rpc") -}}
{{- $sans := printf "alt_names=api.iot.cloud.vx-labs.net" -}}
{{- $path := printf "pki/issue/grpc" -}}
{{ with secret $path $cn $ipsans $sans "ttl=48h" }}{{ .Data.certificate }}{{ end }}
EOH
      }

      template {
        change_mode = "restart"
        destination = "local/key.pem"
        splay       = "1h"

        data = <<EOH
{{- $cn := printf "common_name=%s" (env "NOMAD_ALLOC_ID") -}}
{{- $ipsans := printf "ip_sans=%s" (env "NOMAD_IP_rpc") -}}
{{- $sans := printf "alt_names=api.iot.cloud.vx-labs.net" -}}
{{- $path := printf "pki/issue/grpc" -}}
{{ with secret $path $cn $ipsans $sans "ttl=48h" }}{{ .Data.private_key }}{{ end }}
EOH
      }

      template {
        change_mode = "restart"
        destination = "local/ca.pem"
        splay       = "1h"

        data = <<EOH
{{- $cn := printf "common_name=%s" (env "NOMAD_ALLOC_ID") -}}
{{- $ipsans := printf "ip_sans=%s" (env "NOMAD_IP_rpc") -}}
{{- $sans := printf "alt_names=api.iot.cloud.vx-labs.net" -}}
{{- $path := printf "pki/issue/grpc" -}}
{{ with secret $path $cn $ipsans $sans "ttl=48h" }}{{ .Data.issuing_ca }}{{ end }}
EOH
      }

      config {
        logging {
          type = "fluentd"

          config {
            fluentd-address = "localhost:24224"
            tag             = "alveoli"
          }
        }

        image = "${service_image}:${service_version}"
        args = [
          "--use-vault",
          "--tls-cn", "api.iot.cloud.vx-labs.net",
        ]
        force_pull = true

        port_map {
          http     = 8080
          metrics = 8089
        }
      }

      resources {
        cpu    = 200
        memory = 256

        network {
          mbits = 10
          port "http" {}
          port "metrics" {}
        }
      }


      service {
        name = "alveoli"
        port = "http"
        tags = [
          "http",
          "${service_version}",
          "traefik.enable=true",
          "traefik.tcp.routers.apitls.rule=HostSNI(`api.iot.cloud.vx-labs.net`)",
          "traefik.tcp.routers.apitls.entrypoints=https",
          "traefik.tcp.routers.apitls.service=alveoli",
          "traefik.tcp.routers.apitls.tls",
          "traefik.tcp.routers.apitls.tls.passthrough=true",
        ]

        check {
          type     = "tcp"
          port     = "http"
          interval = "30s"
          timeout  = "2s"
        }
      }
    }
  }
}
