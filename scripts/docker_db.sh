#!/bin/bash
docker run -d \
  --name postgres_yappa \
  -e POSTGRES_USER=yappa \
  -e POSTGRES_PASSWORD=pass \
  -e POSTGRES_DB=yappa-chat \
  -v pgdata:/var/lib/postgresql/data \
  -p 5432:5432 postgres:17 \
  -c shared_preload_libraries=pgcrypto

docker cp scripts/sql/schema.sql postgres_yappa:/schema.sql

docker exec -i postgres_yappa psql -U yappa -d yappa-chat -f /schema.sql