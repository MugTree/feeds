seed-db:
	go run ./data/seed/generate.go -urls ./data/seed/seed.csv -db ./data/feedreader.db

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

production-build-app:
	templ generate && CC=x86_64-linux-musl-gcc CXX=x86_64-linux-musl-g++ GOARCH=amd64 GOOS=linux CGO_ENABLED=1 go build -ldflags "-linkmode external -extldflags -static" -o ./bin/metrics.amd64 .
