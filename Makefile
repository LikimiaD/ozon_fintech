download:
	go mod download

local_build:
	go build -o main .
	./main

docker:
	docker-compose up --build

generate_code:
	go get github.com/99designs/gqlgen/codegen/config@v0.17.47
	go get github.com/99designs/gqlgen/internal/imports@v0.17.47
	go get github.com/99designs/gqlgen@v0.17.47
	go run github.com/99designs/gqlgen generate

generate_docs:
	npm install -g spectaql
	spectaql spectaql-config.yml
