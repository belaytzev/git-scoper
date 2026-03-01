package main

import (
	"sync"
)

// Result holds the outcome of applying config to a single repo.
type Result struct {
	Path string
	Err  error
}

// runAll applies cfg to each repo path using a pool of workers goroutines.
func runAll(repos []string, cfg *Config, workers int) []Result {
	if len(repos) == 0 {
		return nil
	}

	jobs := make(chan string, len(repos))
	for _, r := range repos {
		jobs <- r
	}
	close(jobs)

	out := make(chan Result, len(repos))

	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for repo := range jobs {
				out <- Result{Path: repo, Err: applyConfig(repo, cfg.Name, cfg.Email)}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(out)
	}()

	var results []Result
	for r := range out {
		results = append(results, r)
	}
	return results
}
