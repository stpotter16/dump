shell:
	nix develop -c $$SHELL


server/build:
	./dev-scripts/build-server.sh

server/run: server/build
	./tmp/server

server/live:
	./dev-scripts/serve.sh

server/deploy:
	./dev-scripts/deploy.sh


secrets/hmac:
	xxd -l32 /dev/urandom | xxd -r -ps | base64 | tr -d = | tr + - | tr / _

secrets/gcm:
	openssl rand -base64 32


lint/go:
	./dev-scripts/check-go.sh

lint/shell:
	./dev-scripts/check-shell.sh

lint/sql:
	./dev-scripts/check-sql.sh

lint/frontend:
	./dev-scripts/check-frontend.sh


test/go:
	go test ./...

