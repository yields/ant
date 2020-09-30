
<br>
<br>
<br>

<p align=center>
  ant is a minimal, idiomatic crawler for Go.
</p>

<br>
<br>
<br>

<h1 align=center>
  <a href="https://github.com/yields/ant/workflows/test">
    <img src="https://github.com/yields/ant/workflows/test/badge.svg?event=push" />
  </a>
  <a href="https://pkg.go.dev/github.com/yields/ant">
    <img src="https://pkg.go.dev/badge/github.com/yields/ant" />
  </a>
  <a href="https://goreportcard.com/report/github.com/yields/ant">
    <img src="https://goreportcard.com/badge/github.com/yields/ant" />
  </a>
</h1>

<br>
<br>
<br>


### Synopsis

  Ant is a small web crawler for Go, it aims to follow the `net/http` package
  in style, it aims to be an idiomatic crawler.

  ```go
  var repo struct {
    About string `css:".f4.mt3"`
  }

  page.Scan(&repo)
  fmt.Println(repo.About) // => Modern Web Crawler for Go
  ```

<br>

### Features

  - [x] Polite, follows `robots.txt` automatically.
  - [x] Concurrent.
  - [x] Declarative data extraction.
  - [x] Rate limits.
  - [x] URL matchers.
  - [x] Pluggable.

<br>

### Status

  Note that this project is still a work in progress, please use go modules
  when using it.

  The plan is to add many features that are necessary to crawl webpages:

  - [ ] `antdb` package which implements persistent data-structures.
  - [ ] `antredis` for distributed crawlers.
  - [ ] `Cache` interface to cache webpages.
  - [ ] `antcdp` chrome headless `ant.Fetcher` implementation.
