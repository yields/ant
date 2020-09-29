
headless-image := chromedp/headless-shell:stable
pkg  := .
func := .

deps:
	@go get -d -t ./...
	@docker pull $(headless-image)

test:
	@go test --cover --timeout 5s --race ./...

test.cdp:
	@GOOS=linux GOARCH=amd64 go test -c ./antcdp
	@mv antcdp.test antcdp
	@docker run --rm \
		--volume=$(PWD)/antcdp:/antcdp \
		--entrypoint=/antcdp/antcdp.test \
		--workdir=/antcdp \
		--env=HEADLESS_SHELL=/headless-shell/headless-shell \
		$(headless-image) \
		--test.timeout 10s

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
	rm -fr *.test *.prof *.cover *.trace antcdp/*.test
