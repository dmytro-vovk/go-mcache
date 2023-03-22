# go-mcache
[![.github/workflows/ci.yaml](https://github.com/dmytro-vovk/go-mcache/actions/workflows/ci.yaml/badge.svg)](https://github.com/dmytro-vovk/go-mcache/actions/workflows/ci.yaml)
[![Coverage Status](https://coveralls.io/repos/github/dmytro-vovk/go-mcache/badge.svg)](https://coveralls.io/github/dmytro-vovk/go-mcache)
[![Go Report Card](https://goreportcard.com/badge/github.com/dmytro-vovk/go-mcache)](https://goreportcard.com/report/github.com/dmytro-vovk/go-mcache)
[![Reference](https://pkg.go.dev/badge/github.com/dmytro-vovk/go-mcache.svg)](https://pkg.go.dev/github.com/dmytro-vovk/go-mcache)

Yet another in-memory cache library with expiration.

Uses generics and does not use periodic check for expired items.
Instead, it uses internal ordered queue to range items in order of expiration and just sleeps until it is time to expire an item.
thus having minimal computational overhead.

## Installation

```sh
go get github.com/dmytro-vovk/go-mcache
```

## Usage

```go
// Create a new instance with string keys and int values
c := mcache.New[string, int]()

// Set couple values
c.Set("one", 1, time.Minute)
c.Set("two", 2, time.Hour)

// Get a value
if value, ok := c.Get("one"); ok {
	fmt.Printf("Got value: %v", value)
}

// Get value and delete it from the cache
if value, ok := GetAndDelete("two"); ok {
    fmt.Printf("Got and deleted value: %v", value)
}

// Try getting non-existing value
if _, ok := Get("two"); !ok {
    fmt.Print("Value no longer cached")
}

// Set different value for the key keeping the same TTL
if c.Update("one", 101) {
    fmt.Print("Value was updated")
}

// Replace the value getting the previous one
if value, ok := c.Swap("one", 202); ok {
    fmt.Printf("Previous value was %v", value)
}

// Delete a value
if c.Delete("one") {
    fmt.Print("The value is deleted")
}

// Change the value TTL 
if c.Refresh("one", time.Hour) {
    fmt.Print("TTL for the key is set to one hour")
}

// Let's see how many values we have
fmt.Printf("We have %d values", c.Len())

// Let's free some memory by evicting some data
if evicted := c.Evict(10); evicted > 0 {
    fmt.Printf("Evicted %d values", evicted)
}

// Value expires
c.Set("gone fast", 1000, time.Millisecond)
time.Sleep(time.Millisecond)
if _, ok := c.Get("gone fast"); !ok {
    fmt.Print("the value is gone!")
}
```
