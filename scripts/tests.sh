#!/bin/bash

# Test script for Docker proxy
SOCKET="/var/run/docker.sock"  # Change this to the socket you want to test

docker_api_call() {
    local method=$1
    local endpoint=$2
    curl -s -X $method --unix-socket $SOCKET "http://localhost/v1.46$endpoint"
}

echo "Testing container list:"
docker_api_call GET "/containers/json" | jq .

test_container() {
    local container_id=$1
    
    echo "Testing inspect for container $container_id:"
    docker_api_call GET "/containers/$container_id/json" | jq .

    echo "Testing logs for container $container_id:"
    docker_api_call GET "/containers/$container_id/logs?stdout=1&stderr=1"

    echo "Testing stats for container $container_id:"
    docker_api_call GET "/containers/$container_id/stats?stream=false" | jq .

    echo "Testing top for container $container_id:"
    docker_api_call GET "/containers/$container_id/top" | jq .

    echo "Testing changes for container $container_id:"
    docker_api_call GET "/containers/$container_id/changes" | jq .
}

first_container=$(docker_api_call GET "/containers/json" | jq -r '.[0].Id')

if [ -n "$first_container" ]; then
    test_container $first_container
else
    echo "No containers found. Make sure you have containers running with the correct label."
fi

echo "Testing non-existent container:"
docker_api_call GET "/containers/nonexistent/json"

echo "Testing container creation (should be denied):"
docker_api_call POST "/containers/create" -H "Content-Type: application/json" -d '{"Image": "nginx"}'