#!/bin/sh
for i in {1..10000}; do curl -XPOST --data-binary @payload.json localhost:9143/collector -H Content-Type:application/json;echo $i; done

