
<br>
<br>
<br>

<p align=center>
  ant (<em>alpha</em>) is a web crawler for Go.
</p>

<br>
<br>
<br>

<p align=center>
  <a href="https://github.com/yields/ant/workflows/test">
    <img src="https://github.com/yields/ant/workflows/test/badge.svg?event=push" />
  </a>
  <a href="https://pkg.go.dev/github.com/yields/ant">
    <img src="https://pkg.go.dev/badge/github.com/yields/ant" />
  </a>
  <a href="https://goreportcard.com/report/github.com/yields/ant">
    <img src="https://goreportcard.com/badge/github.com/yields/ant" />
  </a>
</p>

<br>
<br>
<br>


<br>

##### Polite

  The crawler automatically fetches and caches `robots.txt`, making sure that
  it never causes issues to small website owners. Of-course you can disable
  this behavior [easily]().

  ```go
  eng, err := ant.NewEngine(ant.EngineConfig{
    Impolite: true,
  })
  eng.Run(ctx)
  ```

<br>

##### Concurrent

  The crawler maintains a configurable amount of "worker" goroutines that read
  URLs off the queue, and spawn a goroutine for each URL.

  Depending on your configuration, you may want to increase the number of workers
  to speed up URL reads, of-course if you don't have enough resources you can reduce
  the number of workers too.

  ```go
  eng, err := ant.NewEngine(ant.EngineConfig{
    // Spawn 5 worker goroutines that dequeue
    // URLs and spawn a new goroutine for each URL.
    Workers: 5,
  })
  eng.Run(ctx)
  ```

<br>

##### Declarative

  The package includes functions that can scan data from the page into your structs
  or slice of structs, this allows you to reduce the noise and complexity in your source-code.

  You can also use a [jQuery-like API]() that allows you to scrape complex HTML pages if needed.

  ```go

  var data struct { Title string `css:"title"` }
  page, _ := ant.Fetch(ctx, "https://apple.com")
  page.Scan(&data)
  data.Title // => Apple
  ```

<br>

##### Ratelimits

  The package includes a powerful `ant.Limiter` interface that allows you to
  define rate-limits per URL, of-course there are also some built-in limiters.

  ```go
  ant.Limit(1) // 1 rps on all URLs.
  ant.LimitHostname(5, "amazon.com") // 5 rps on amazon.com hostname.
  ant.LimitPattern(5, "amazon.com.*") // 5 rps on all amazon.co.
  ant.LimitRegexp(5, "^apple.com\/iphone\/*") // 2 rps on URLs that match.
  ```

<br>

##### Matchers

  Another powerful interface is `ant.Matcher` which allows you to define URL
  matchers, the matchers are called before URLs are queued.

  ```go
  ant.MatchHostname("amazon.com") // scrape amazon.com URLs only.
  ant.MatchPattern("amazon.com/help/*")
  ant.MatchRegexp("amazon\.com\/help/.+")
  ```

<br>

##### Robust

  The crawl engine automatically retries any errors that implement `Temporary()`
  error that returns true.

  Becuase the standard library returns errors that implement that interface
  the engine will retry most temporary network and HTTP errors.

  ```go
  eng, err := ant.NewEngine(ant.EngineConfig{
    Scraper: myscraper{},
    MaxAttempts: 5,
  })

  // Blocks until one of the following is true:
  //
  // 1. No more URLs to crawl (the scraper stops returning URLs)
  // 2. A non-temporary error occured.
  // 3. MaxAttempts was reached.
  //
  err = eng.Run(ctx)
  ```

<br>

##### Built-in Scrapers

  The whole point of scraping is to extract data from websites into a machine readable
  format such as CSV or JSON, ant comes with built-in scrapers to make this ridiculously
  easy, here's a full cralwer that extracts quotes into stdout.

  ```Go
  // Describe how a quote should be extracted.
  type Quote struct {
    Text string   `css:".text"`
    By   string   `css:".author"`
    Tags []string `css:".tag"`
  }

  // A page may have many quotes.
  type Page struct {
    Quotes []Quote `css:".quote"`
  }

  // Where we want to fetch quotes from.
  const host = "quotes.toscrape.com"

  // Initialize the engine with a built-in scraper
  // that receives a type and extract data into an io.Writer.
  eng, err := ant.NewEngine(ant.EngineConfig{
    Scraper: ant.JSON(Page{}, os.Stdout),
    Matcher: ant.MatchHostname(host),
  })

  // Block until there are no more URLs to scrape.
  eng.Run(ctx, "http:// "+ host)
  ```

<br>

##### Testing

  `anttest` package makes it easy to test your scraper implementation
  it fetches a page by a URL, caches it in the OS's temporary directory and re-uses it.

  The func depends on the file's modtime, the file expires daily, you can adjust
  the TTL by setting `antttest.FetchTTL`.

  ```Go
  // Fetch calls `t.Fatal` on errors.
  page := anttest.Fetch(t, "https://apple.com")
  _, err := myscraper.Scrape(ctx, page)
  assert.NoError(err)
  ```

<br>
<br>
