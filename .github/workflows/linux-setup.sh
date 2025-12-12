#!/usr/bin/env bash

set -o errexit

# Basic tools

set -x

if [ ! -z "${DOCKERHUB_PULL_USERNAME:-}" ]; then
  set +x
  echo "${DOCKERHUB_PULL_PASSWORD}" | docker login --username "${DOCKERHUB_PULL_USERNAME}" --password-stdin
  set -x
fi

sudo apt-get update -qq >/dev/null
sudo apt-get install -y -qq build-essential expect libnss3-tools libcurl4-gnutls-dev postgresql-client >/dev/null

curl -sSL https://ngrok-agent.s3.amazonaws.com/ngrok.asc \
  | sudo tee /etc/apt/trusted.gpg.d/ngrok.asc >/dev/null \
  && echo "deb https://ngrok-agent.s3.amazonaws.com bookworm main" \
  | sudo tee /etc/apt/sources.list.d/ngrok.list \
  && sudo apt-get update -qq >/dev/null \
  && sudo apt-get install -y -qq ngrok

# Install cloudflared
sudo mkdir -p --mode=0755 /usr/share/keyrings
curl -fsSL https://pkg.cloudflare.com/cloudflare-main.gpg \
  | sudo tee /usr/share/keyrings/cloudflare-main.gpg >/dev/null \
  && echo 'deb [signed-by=/usr/share/keyrings/cloudflare-main.gpg] https://pkg.cloudflare.com/cloudflared any main' \
  | sudo tee /etc/apt/sources.list.d/cloudflared.list \
  && sudo apt-get update -qq >/dev/null \
  && sudo apt-get install -y -qq cloudflared

if [[ ${DDEV_TEST_PODMAN_ROOTLESS:-} == "true" ]]; then
  echo "Setting up podman-rootless"
  sudo systemctl disable --now docker.service docker.socket
  sudo rm -f /var/run/docker.sock
  sudo apt-get remove -y podman crun
  sudo apt-get install -y fuse-overlayfs
  brew install podman >/dev/null
  hash -r
  # Enable ports below 1024
  sudo mkdir -p /etc/sysctl.d
  echo 'net.ipv4.ip_unprivileged_port_start=0' | sudo tee -a /etc/sysctl.d/60-rootless.conf
  sudo sysctl -p /etc/sysctl.d/60-rootless.conf
  # Create systemd unit files
  mkdir -p ~/.config/systemd/user
  # Create podman.socket
  cat > ~/.config/systemd/user/podman.socket <<'EOF'
[Unit]
Description=Podman API Socket
Documentation=man:podman-system-service(1)

[Socket]
ListenStream=%t/podman/podman.sock
SocketMode=0660

[Install]
WantedBy=sockets.target
EOF
  # Create podman.service (adapted for Homebrew path)
  cat > ~/.config/systemd/user/podman.service <<'EOF'
[Unit]
Description=Podman API Service
Requires=podman.socket
After=podman.socket
Documentation=man:podman-system-service(1)
StartLimitIntervalSec=0

[Service]
Delegate=true
Type=exec
KillMode=process
Environment=LOGGING="--log-level=info"
ExecStart=/home/linuxbrew/.linuxbrew/bin/podman $LOGGING system service

[Install]
WantedBy=default.target
EOF
  # Reload systemd
  systemctl --user daemon-reload
  # Set DNS
  mkdir -p ~/.config/containers/containers.conf.d
  cat << 'EOF' > ~/.config/containers/containers.conf.d/dns.conf
[containers]
dns_servers = ["1.1.1.1", "1.0.0.1"]
EOF
  # https://github.com/containers/podman/blob/main/docs/tutorials/performance.md#choosing-a-storage-driver
  cat << 'EOF' > ~/.config/containers/storage.conf
[storage]
driver = "overlay"
[storage.options.overlay]
mount_program = "/usr/bin/fuse-overlayfs"
EOF
  # Enable and start the socket
  systemctl --user enable --now podman.socket
  # Try several times, it can return "failed to reexec: Permission denied"
  podman info --format '{{.Host.RemoteSocket.Path}}' || podman info --format '{{.Host.RemoteSocket.Path}}' || true
  # Switch to the podman context
  docker context create podman-rootless --docker host="unix://$(podman info --format '{{.Host.RemoteSocket.Path}}')"
  docker context use podman-rootless
  echo "Verifying podman-rootless setup"
  cat /etc/subuid
  cat /etc/subgid
  podman run --rm ddev/ddev-utilities cat /etc/resolv.conf
  podman info
  podman version
elif [[ "${DDEV_TEST_PODMAN_ROOT:-}" == "true" ]]; then
  echo "Setting up podman-root"
  sudo systemctl disable --now docker.service docker.socket
  sudo rm -f /var/run/docker.sock
  sudo apt-get remove -y podman crun
  brew install podman >/dev/null
  hash -r
  # Configure podman to run in root mode
  # Create tmpfiles.d entry for /run/podman
  sudo tee /etc/tmpfiles.d/podman.conf > /dev/null <<EOF
d /run/podman 0770 root docker
EOF
  sudo systemd-tmpfiles --create
  # Create systemd unit files
  sudo mkdir -p /etc/systemd/system
  # Create podman.socket
  sudo tee /etc/systemd/system/podman.socket > /dev/null <<EOF
[Unit]
Description=Podman API Socket
Documentation=man:podman-system-service(1)

[Socket]
ListenStream=%t/podman/podman.sock
SocketMode=0660

[Install]
WantedBy=sockets.target
EOF
  # Create podman.service (adapted for Homebrew path)
  sudo tee /etc/systemd/system/podman.service > /dev/null <<EOF
[Unit]
Description=Podman API Service
Requires=podman.socket
After=podman.socket
Documentation=man:podman-system-service(1)
StartLimitIntervalSec=0

[Service]
Delegate=true
Type=exec
KillMode=process
Environment=LOGGING="--log-level=info"
ExecStart=/home/linuxbrew/.linuxbrew/bin/podman \$LOGGING system service

[Install]
WantedBy=default.target
EOF
  # Create override for podman.socket to set user:group
  sudo mkdir -p /etc/systemd/system/podman.socket.d
  sudo tee /etc/systemd/system/podman.socket.d/override.conf > /dev/null <<EOF
[Socket]
SocketMode=0660
SocketUser=root
SocketGroup=docker
EOF
  # Reload systemd
  sudo systemctl daemon-reload
  # Enable and start the socket
  sudo systemctl enable --now podman.socket
  # Try several times, it can return "failed to reexec: Permission denied"
  sudo /home/linuxbrew/.linuxbrew/bin/podman info --format '{{.Host.RemoteSocket.Path}}' || sudo /home/linuxbrew/.linuxbrew/bin/podman info --format '{{.Host.RemoteSocket.Path}}' || true
  # Switch to the podman context
  docker context create podman --docker host="unix://$(sudo /home/linuxbrew/.linuxbrew/bin/podman info --format '{{.Host.RemoteSocket.Path}}')"
  docker context use podman
  echo "Verifying podman-root setup"
  sudo /home/linuxbrew/.linuxbrew/bin/podman run --rm ddev/ddev-utilities cat /etc/resolv.conf
  sudo /home/linuxbrew/.linuxbrew/bin/podman info
  sudo /home/linuxbrew/.linuxbrew/bin/podman version
elif [[ "${DDEV_TEST_DOCKER_ROOTLESS:-}" == "true" ]]; then
  echo "Setting up docker-rootless"
  sudo systemctl disable --now docker.service docker.socket
  sudo rm -f /var/run/docker.sock
  # Enable ports below 1024
  sudo mkdir -p /etc/sysctl.d
  echo 'net.ipv4.ip_unprivileged_port_start=0' | sudo tee -a /etc/sysctl.d/60-rootless.conf
  sudo sysctl -p /etc/sysctl.d/60-rootless.conf
  # Configure AppArmor for rootlesskit
  # Source: https://github.com/ScribeMD/rootless-docker/pull/402
  abi4_version="$(find /etc/apparmor.d/abi -maxdepth 1 -name '4.*' -printf '%f\n' | sort -nr | head -1)"
  filename=$(echo $HOME/bin/rootlesskit | sed -e s@^/@@ -e s@/@.@g)
  sudo tee /etc/apparmor.d/${filename} > /dev/null <<EOF
abi <abi/${abi4_version}>,

include <tunables/global>

"$HOME/bin/rootlesskit" flags=(unconfined) {
userns,

include if exists <local/${filename}>
}
EOF
  sudo systemctl restart apparmor.service
  # Allow loopback https://github.com/moby/moby/issues/47684#issuecomment-2166149845
  mkdir -p ~/.config/systemd/user/docker.service.d
  cat << 'EOF' > ~/.config/systemd/user/docker.service.d/override.conf
[Service]
Environment="DOCKERD_ROOTLESS_ROOTLESSKIT_DISABLE_HOST_LOOPBACK=false"
EOF
  # Install rootless docker
  curl -fsSL https://get.docker.com/rootless | sh
  cat /etc/subuid
  cat /etc/subgid
fi

# Without this .curlrc CircleCI linux image doesn't respect mkcert certs
echo "capath=/etc/ssl/certs/" >>~/.curlrc

source ~/.bashrc

brew tap bats-core/bats-core >/dev/null
brew tap ddev/ddev >/dev/null
for item in bats-core ddev docker-compose ghr golangci-lint bats-assert bats-file bats-support; do
    brew install $item >/dev/null || brew upgrade $item >/dev/null
done

mkcert -install

# Show info to simplify debugging
docker info
docker version
lsb_release -a
