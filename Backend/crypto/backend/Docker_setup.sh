#!/bin/bash

docker build -t astra-db .

docker run -d -p 8002:8002 --name AstraDB \
  -e ASTRA_DB_ID="$ASTRA_DB_ID" \
  -e ASTRA_DB_APPLICATION_TOKEN="$ASTRA_DB_APPLICATION_TOKEN" \
  astra-db
