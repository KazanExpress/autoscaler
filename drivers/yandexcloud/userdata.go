package yandexcloud

import "github.com/drone/autoscaler/drivers/internal/userdata"

var userdataT = userdata.Parse(`#cloud-config
#cloud-config
write_files:
  - path: /etc/systemd/system/docker.service.d/override.conf
    permissions: '0644'
    content: |
    [Service]
    ExecStart=
    ExecStart=/usr/bin/dockerd
  - path: /etc/default/docker
    permissions: '0644'
    content: |
    DOCKER_OPTS=""
  - path: /etc/docker/daemon.json
    permissions: '0644'
    content: |
    {
      "hosts": [ "0.0.0.0:2376", "unix:///var/run/docker.sock" ],
      "tls": true,
      "tlsverify": true,
      "tlscacert": "/etc/docker/ca.pem",
      "tlscert": "/etc/docker/server-cert.pem",
      "tlskey": "/etc/docker/server-key.pem"
    }
  - path: /etc/docker/ca.pem
    permissions: '0644'
    encoding: b64
    content: {{ .CACert | base64 }}
  - path: /etc/docker/server-cert.pem
    permissions: '0644'
    encoding: b64
    content: {{ .TLSCert | base64 }}
  - path: /etc/docker/server-key.pem
    permissions: '0644'
    encoding: b64
    content: {{ .TLSKey | base64 }}

runcmd:
  - sudo systemctl daemon-reload
  - sudo systemctl restart docker.service
`)
