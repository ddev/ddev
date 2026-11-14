#!/usr/bin/env bash
# Does `docker compose build` then `up -d` recreate a service whose
# dockerfile_inline changed, while another service's Dockerfile also changed?
# That is what DDEV's TestStartProfileImages does. Docker Engine prints
# inline-<n> matching each round. Run with DOCKER_CONTEXT=socktainer or orbstack.
set -u
d="${HOME}/tmp/inline-recreate"
p=inlinerec
rounds="${1:-4}"
rm -rf "${d}" && mkdir -p "${d}/ctx" && cd "${d}" || exit 1
printf 'FROM busybox:stable\nRUN echo x1 > /x1.txt\n' > ctx/Dockerfile
cat > compose.yaml <<'EOF'
services:
  x1:
    build:
      context: ./ctx
    command: tail -f /dev/null
  x2:
    build:
      context: ./ctx
      dockerfile_inline: |
        FROM busybox:stable
        RUN echo inline-0 > /inline.txt
    command: tail -f /dev/null
EOF
docker compose -p "${p}" down -v --rmi all >/dev/null 2>&1
docker compose -p "${p}" build >/dev/null 2>&1
docker compose -p "${p}" up -d >/dev/null 2>&1
for n in $(seq 1 "${rounds}"); do
  sed -i '' "s/inline-[0-9]*/inline-${n}/" compose.yaml
  echo "RUN touch /changed-${n}" >> ctx/Dockerfile
  before=$(docker image inspect "${p}-x2" -f "{{.Id}}")
  docker compose -p "${p}" build > build.log 2>&1
  after=$(docker image inspect "${p}-x2" -f "{{.Id}}")
  if [ "${before}" = "${after}" ]; then echo "  x2 image unchanged by the build; build.log:"; sed "s/^/    /" build.log; fi
  up=$(docker compose -p "${p}" up -d 2>&1 | grep -E 'x[12]-1 +(Recreated|Running)' | tr -s ' \n' ' ')
  echo "round ${n}: x2 has $(docker exec "${p}-x2-1" cat /inline.txt) | ${up}"
  echo "  x2 label=$(docker inspect "${p}-x2-1" -f '{{index .Config.Labels "com.docker.compose.image"}}') image=$(docker image inspect "${p}-x2" -f '{{.Id}}')"
done
docker compose -p "${p}" down -v --rmi all >/dev/null 2>&1
