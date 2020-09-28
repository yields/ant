
pkg  := .
func := .

deps:
	@go get -d -t ./...

test:
	@go test --cover --timeout 5s --race ./...

bench:
	@go test \
		--run=Bench \
		--bench=$(func) \
		--benchmem \
		--blockprofile=block.prof \
		--cpuprofile=cpu.prof \
		--memprofile=mem.prof \
		$(pkg)

prof.%:
	@go tool pprof --http :8080 ant.test $*.prof

cover:
	@go test --coverprofile test.cover ./...
	@go tool cover --html=test.cover

clean:
	rm -fr *.test *.prof *.cover
