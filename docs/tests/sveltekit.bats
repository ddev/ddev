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
  run -0 mkdir -p ${SVELTEKIT_SITENAME}
  cd ${SVELTEKIT_SITENAME}

  run -0 ddev config --project-type=generic --webserver-type=generic
  DDEV_DEBUG=true run -0 ddev start -y

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
  run -0 ddev mutagen sync
  assert_file_exist .ddev/config.sveltekit.yaml

  run -0 ddev exec "yes | npx sv create --template=demo --types=ts --no-add-ons --no-install ."

  run -0 ddev exec "sed -i -e 's#adapter-auto#adapter-node#' -e 's#^export default defineConfig({#&\n\tserver: { port: 3000, allowedHosts: true },#' vite.config.ts"

  run -0 ddev npm install @sveltejs/adapter-node
  run -0 ddev npm install
  run -0 ddev npm run build
  DDEV_DEBUG=true run -0 ddev restart -y

  # web_extra_exposed_ports only takes effect after this restart, so the
  # generic webserver has no primary URL to report until now (see
  # GetPrimaryRouterHTTP(S)Port in pkg/ddevapp/ddevapp.go).
  _extra_info

  # ddev launch
  DDEV_DEBUG=true run -0 ddev launch
  assert_output "FULLURL ${PRIMARY_URL}"
  # validate running project
  _assert_page_contains ${PRIMARY_URL} "to your new"
}
