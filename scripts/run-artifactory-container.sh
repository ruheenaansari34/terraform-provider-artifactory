#!/usr/bin/env bash

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" > /dev/null && pwd )"

set -euf

docker run -i -t -d --rm -v "${SCRIPT_DIR}/artifactory.lic:/artifactory_extra_conf/artifactory.lic:ro" \
  -p8081:8081 -p8082:8082 -p8080:8080 releases-docker.jfrog.io/jfrog/artifactory-pro:7.24.3

echo "Waiting for Artifactory to start"
echo "Sleep"
sleep 60
docker ps
curl -sf -u admin:password http://localhost:8081/artifactory/api/system/licenses/

#echo "Waiting for Artifactory to start"
#until curl -sf -u admin:password http://localhost:8081/artifactory/api/system/licenses/; do
#    printf '.'
#    sleep 4
#done
#echo ""