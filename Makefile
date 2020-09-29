
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
		$(pkg)

prof.%:
	@go test --run=Bench --bench=$(func) --$*profile=$*.prof
	@go tool pprof --http :8080 ant.test $*.prof

trace:
	@go test --run=Bench --bench=$(func) --trace ant.trace $(pkg)
	@go tool trace ant.trace

cover:
	@go test --coverprofile test.cover ./...
	@go tool cover --html=test.cover

clean:
	rm -fr *.test *.prof *.cover *.trace
