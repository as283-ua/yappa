.PHONY: cert_server cacert proto cert_server_ca_tls cert_server_server \
	docker_db docker_clean \
	clean  all \
	clean_session deep_clean \
	bin

api/gen/ca/ca.pb.go: api/proto/ca.proto
	mkdir -p api/gen/ca
	protoc -I=api/proto --go_out=api/gen/ca --go_opt=paths=source_relative $<

api/gen/server/server.pb.go: api/proto/server.proto
	mkdir -p api/gen/server
	protoc -I=api/proto --go_out=api/gen/server --go_opt=paths=source_relative $<

api/gen/client/client.pb.go: api/proto/client.proto
	mkdir -p api/gen/client
	protoc -I=api/proto --go_out=api/gen/client --go_opt=paths=source_relative $<

proto: api/gen/ca/ca.pb.go api/gen/server/server.pb.go api/gen/client/client.pb.go

cert_server:
	@if [ -z "$(ARG)" ]; then \
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

cert_server_ca_tls:
	$(MAKE) cert_server ARG=ca_tls

cert_server_server:
	$(MAKE) cert_server ARG=server

certs/ca/ca.crt certs/ca/ca.key:
	mkdir -p certs/ca
	openssl genpkey -algorithm EC -pkeyopt ec_paramgen_curve:P-384 -out certs/ca/ca.key
	openssl req -x509 -new -key certs/ca/ca.key -sha256 -days 3650 -out certs/ca/ca.crt -config certs/ca.cnf -extensions v3_ca

cacert: certs/ca/ca.crt certs/ca/ca.key

cert_test_ok: certs/ca/ca.crt certs/ca/ca.key
	mkdir -p test/assets/certs/test_ok
	rm -rf test/assets/certs/test_ok/*
	openssl ecparam -genkey -name secp384r1 | openssl pkcs8 -topk8 -nocrypt -out test/assets/certs/test_ok/test_ok.key
	openssl req -new -key test/assets/certs/test_ok/test_ok.key \
		-out test/assets/certs/test_ok/test_ok.csr \
		-subj "/CN=test_ok"
	openssl x509 -req -days 365 \
		-in test/assets/certs/test_ok/test_ok.csr \
		-CA certs/ca/ca.crt -CAkey certs/ca/ca.key -CAcreateserial \
		-out test/assets/certs/test_ok/test_ok.crt \
		-extensions req_ext -extfile certs/server.cnf
	rm test/assets/certs/test_ok/test_ok.csr
	openssl verify -CAfile certs/ca/ca.crt test/assets/certs/test_ok/test_ok.crt

cert_test_bad:
	mkdir -p test/assets/certs/test_bad
	rm -rf test/assets/certs/test_bad/*
	openssl genpkey -algorithm EC -pkeyopt ec_paramgen_curve:P-384 -out test/assets/certs/test_bad/test_bad.key
	openssl req -new -x509 -key test/assets/certs/test_bad/test_bad.key -sha256 -days 365 -out test/assets/certs/test_bad/test_bad.crt -config certs/server.cnf -extensions req_ext
	openssl verify -CAfile certs/ca/ca.crt test/assets/certs/test_bad/test_bad.crt || true

sqlc: scripts/sql/queries.sql scripts/sql/schema.sql
	sqlc generate

all: cacert cert_server_ca_tls cert_server_server cert_test_ok cert_test_bad proto sqlc

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

clean:
	rm -f bin/*

docker_clean:
	docker cp scripts/sql/schema.sql postgres_yappa:/schema.sql
	docker exec -i postgres_yappa psql -U yappa -d yappa-chat -f /schema.sql

clean_session:
	rm -f logs/cli/* logs/ca/* logs/serv/*
	rm -f **.data

deep_clean: clean
	rm -rf certs/ca/* certs/ca_tls/* certs/client/* certs/peer/* certs/server/* test/assets/certs/*

BIN_DIR := bin
CLIENT_BIN := $(BIN_DIR)/yappa
SERVER_BIN := $(BIN_DIR)/yappad
CA_BIN     := $(BIN_DIR)/yappacad

GO_SOURCES := $(shell find . -type f -name '*.go')

bin: proto sqlc $(CLIENT_BIN) $(SERVER_BIN) $(CA_BIN)

$(CLIENT_BIN): $(GO_SOURCES)
	mkdir -p $(BIN_DIR)
	go build -o $@ ./cmd/client

$(SERVER_BIN): $(GO_SOURCES)
	mkdir -p $(BIN_DIR)
	go build -o $@ ./cmd/server

$(CA_BIN): $(GO_SOURCES)
	mkdir -p $(BIN_DIR)
	go build -o $@ ./cmd/ca