#!/usr/bin/env bats

setup() {
  PROJNAME=my-sveltekit-site
  load 'common-setup'
  _common_setup
}

# executed after each test
teardown() {
  _common_teardown
}

@test "sveltekit quickstart with $(ddev --version)" {
  SVELTEKIT_SITENAME=${PROJNAME}
  run mkdir ${SVELTEKIT_SITENAME} && cd ${SVELTEKIT_SITENAME}
  assert_success

  run ddev config --project-type=generic --webserver-type=generic
  assert_success
  DDEV_DEBUG=true run ddev start -y
  assert_success

  cat <<EOF > .ddev/config.sveltekit.yaml
web_extra_exposed_ports:
    - name: svelte
      container_port: 3000
      http_port: 80
      https_port: 443
web_extra_daemons:
    - name: "sveltekit-demo"
      command: "node build"
      directory: /var/www/html
EOF
  assert_success

  run ddev exec "yes | npx sv create --template=demo --types=ts --no-add-ons --no-install ."
  assert_success

  run ddev exec curl -s -OL https://raw.githubusercontent.com/ddev/test-sveltekit/main/svelte.config.js
  assert_success

  run ddev exec curl -s -OL https://raw.githubusercontent.com/ddev/test-sveltekit/main/vite.config.ts
  assert_success

  run ddev npm install @sveltejs/adapter-node
  assert_success
  run ddev npm install
  assert_success
  run ddev npm run build
  assert_success
  DDEV_DEBUG=true run ddev restart -y
  assert_success

  # Diagnostic: show traefik config files in volume
  run docker exec ddev-router ls -la /mnt/ddev-global-cache/traefik/config/
  echo "# Traefik config files: $output"
  # Diagnostic: show traefik router API response (just router names)
  run docker exec ddev-router curl -s http://127.0.0.1:10999/api/http/routers
  echo "# Traefik routers: $output"

  run bash -c '
  docker ps -q \
    --filter "label=com.ddev.platform=ddev" \
    --filter "label=com.docker.compose.service=web" \
    --filter "label=com.docker.compose.oneoff=False" |
  xargs -r docker inspect --format "{{.Name}} {{.State.Status}} {{if .State.Health}}{{.State.Health.Status}}{{else}}no-health{{end}}"
'
  assert_output --partial "${PROJNAME}-web running healthy"
  assert_success
  echo "# Existing containers: $output" >&3

  # ddev launch
  run bash -c "DDEV_DEBUG=true ddev launch"
  assert_output "FULLURL https://${PROJNAME}.ddev.site"
  assert_success
  # validate running project
  run curl -sfv https://${PROJNAME}.ddev.site
  assert_output --partial "to your new"
  assert_success
}
