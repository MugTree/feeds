drop-data:
	rm feeds.db
	rm feeds.db-*

seed-data:
	touch feeds.db
	goose status
	goose up
	go run ./app/db/seed/generate.go --urls=./app/db/seed/seed.csv --db=./feeds.db

lint:
	golangci-lint run .

run-tests:
	cd www && go test -v && cd ..
  
format-html:
	templ fmt ./app 

dev:
	air  

debug:
	go build -gcflags="all=-N -l" -o ./tmp/server .

