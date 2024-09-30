#!/bin/sh
# UNUSED !
set -e

env | grep DESTINATION_SOCKET | while read line; do
    echo rm -v $(echo $line | cut -d'=' -f2)
done 

exec "$@"