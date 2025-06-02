.PHONY: proto cert_server docker_db clear cacert

proto:
	rm -rf api/gen/*
	mkdir -p api/gen/ca api/gen/server api/gen/client
	protoc -I=api/proto --go_out=api/gen/ca --go_opt=paths=source_relative api/proto/ca.proto
	protoc -I=api/proto --go_out=api/gen/server --go_opt=paths=source_relative api/proto/server.proto
	protoc -I=api/proto --go_out=api/gen/client --go_opt=paths=source_relative api/proto/client.proto

cert_server:
	@if [ "$$#" -ne 1 ]; then \
		echo "Error: Exactly 1 argument is required."; \
		echo "Usage: make cert_server ARG=<name>"; \
		exit 1; \
	fi
	mkdir -p certs/$(ARG)
	rm -rf certs/$(ARG)/*
	openssl ecparam -genkey -name secp384r1 | openssl pkcs8 -topk8 -nocrypt -out certs/$(ARG)/$(ARG).key
	openssl req -new -key certs/$(ARG)/$(ARG).key -out certs/$(ARG)/$(ARG).csr -config certs/server.cnf
	openssl req -noout -text -in certs/$(ARG)/$(ARG).csr | grep -A 1 "Subject Alternative Name"
	openssl x509 -req -days 365 -in certs/$(ARG)/$(ARG).csr -CA certs/ca/ca.crt -CAkey certs/ca/ca.key -CAcreateserial -out certs/$(ARG)/$(ARG).crt -extensions req_ext -extfile certs/server.cnf
	rm certs/$(ARG)/$(ARG).csr
	openssl verify -CAfile certs/ca/ca.crt certs/$(ARG)/$(ARG).crt

docker_db:
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

clear:
	rm -f logs/cli/* logs/ca/* logs/serv/*
	rm -f **.data

cacert:
	rm -f certs/ca/ca.key certs/ca/ca.crt
	mkdir -p certs/ca
	openssl genpkey -algorithm EC -pkeyopt ec_paramgen_curve:P-384 -out certs/ca/ca.key
	openssl req -x509 -new -key certs/ca/ca.key -sha256 -days 3650 -out certs/ca/ca.crt -config certs/ca.cnf -extensions v3_ca
