![test](https://github.com/yields/ant/workflows/test/badge.svg)

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

### Features

  - [x] Polite, follows `robots.txt` automatically.
  - [x] Concurrent.
  - [x] Declarative data extraction.
  - [x] Rate limits.
  - [x] URL matchers.
  - [x] Pluggable.

### Status

  Note that this project is still a work in progress, please use go modules
  when using it.

  The plan is to add many features that are necessary to crawl webpages:

  - [ ] `antdb` package which implements persistent data-structures.
  - [ ] `antredis` for distributed crawlers.
  - [ ] `Cache` interface to cache webpages.
  - [ ] `antcdp` chrome headless `ant.Fetcher` implementation.
