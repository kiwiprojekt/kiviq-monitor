package agent

import (
	"context"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/michal/kiviq/internal/shared"
)

// A wedged Docker daemon (a stats call that never returns on its own) must not
// hang the collection cycle forever. Once the shared context is cancelled, the
// blocked fetch unblocks via ctx.Done() and the receive loop completes, leaving
// the stuck container un-enriched while healthy ones are still filled in.
func TestCollectDockerStatsHonorsContextCancellation(t *testing.T) {
	containers := []shared.DockerContainer{{ID: "slow"}, {ID: "fast"}}

	ctx, cancel := context.WithCancel(context.Background())
	fetch := func(fctx context.Context, id string) (shared.DockerContainer, error) {
		if id == "slow" {
			<-fctx.Done()
			return shared.DockerContainer{}, fctx.Err()
		}
		return shared.DockerContainer{CPUPercent: 7}, nil
	}

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	done := make(chan []shared.DockerContainer, 1)
	go func() { done <- collectDockerStats(ctx, containers, fetch) }()

	select {
	case res := <-done:
		if res[0].CPUPercent != 0 {
			t.Errorf("wedged container should be left un-enriched, got %v", res[0].CPUPercent)
		}
		if res[1].CPUPercent != 7 {
			t.Errorf("healthy container CPU = %v, want 7", res[1].CPUPercent)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("collectDockerStats hung on a wedged fetch")
	}
}

func TestCollectDockerStatsNoDeadlock(t *testing.T) {
	containers := []shared.DockerContainer{
		{ID: "aaa", Name: "c1"},
		{ID: "bbb", Name: "c2"},
		{ID: "ccc", Name: "c3"},
	}

	ch := make(chan dockerStatsResult, len(containers))

	var wg sync.WaitGroup
	for i := range containers {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			if idx == 0 {
				ch <- dockerStatsResult{index: idx, err: true}
				return
			}
			ch <- dockerStatsResult{index: idx, stats: shared.DockerContainer{CPUPercent: 50.0}}
		}(i)
	}

	done := make(chan struct{})
	go func() {
		for range containers {
			r := <-ch
			if !r.err {
				containers[r.index].CPUPercent = r.stats.CPUPercent
			}
		}
		close(done)
	}()

	select {
	case <-done:
		// OK - no deadlock
	case <-time.After(2 * time.Second):
		t.Fatal("deadlock: receive loop did not complete")
	}

	wg.Wait()

	if containers[0].CPUPercent != 0 {
		t.Errorf("failed container should have 0 CPU, got %v", containers[0].CPUPercent)
	}
	if containers[1].CPUPercent != 50.0 {
		t.Errorf("container 1 CPU = %v, want 50.0", containers[1].CPUPercent)
	}
}

func TestCollectDockerStatsAllFail(t *testing.T) {
	containers := []shared.DockerContainer{
		{ID: "aaa", Name: "c1"},
		{ID: "bbb", Name: "c2"},
	}

	ch := make(chan dockerStatsResult, len(containers))

	for i := range containers {
		go func(idx int) {
			ch <- dockerStatsResult{index: idx, err: true}
		}(i)
	}

	done := make(chan struct{})
	go func() {
		for range containers {
			<-ch
		}
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("deadlock when all goroutines fail")
	}
}

func TestCollectDockerStatsNilFetch(t *testing.T) {
	result := collectDockerStats(context.Background(), []shared.DockerContainer{{ID: "a"}}, nil)
	if len(result) != 1 {
		t.Errorf("expected 1 container, got %d", len(result))
	}
}

func TestCollectDockerStatsEmpty(t *testing.T) {
	result := collectDockerStats(context.Background(), nil, nil)
	if len(result) != 0 {
		t.Errorf("expected 0 containers, got %d", len(result))
	}
}

func TestCollectDockerStatsEnrichesInPlace(t *testing.T) {
	containers := []shared.DockerContainer{{ID: "a"}, {ID: "b"}}
	result := collectDockerStats(context.Background(), containers, func(_ context.Context, id string) (shared.DockerContainer, error) {
		return shared.DockerContainer{CPUPercent: 12.5, MemoryPercent: 30}, nil
	})
	for _, c := range result {
		if c.CPUPercent != 12.5 || c.MemoryPercent != 30 {
			t.Errorf("container %s not enriched: %+v", c.ID, c)
		}
	}
}

// The fan-out must never exceed maxDockerStatsConcurrency in-flight calls, so a
// host with many containers cannot flood the Docker daemon.
func TestCollectDockerStatsBoundsConcurrency(t *testing.T) {
	const n = 50
	containers := make([]shared.DockerContainer, n)
	for i := range containers {
		containers[i].ID = string(rune('a' + i%26))
	}

	var mu sync.Mutex
	inFlight, maxSeen := 0, 0

	collectDockerStats(context.Background(), containers, func(_ context.Context, id string) (shared.DockerContainer, error) {
		mu.Lock()
		inFlight++
		if inFlight > maxSeen {
			maxSeen = inFlight
		}
		mu.Unlock()

		time.Sleep(2 * time.Millisecond) // hold the slot so overlap is observable

		mu.Lock()
		inFlight--
		mu.Unlock()
		return shared.DockerContainer{CPUPercent: 1}, nil
	})

	if maxSeen > maxDockerStatsConcurrency {
		t.Errorf("peak concurrency %d exceeded cap %d", maxSeen, maxDockerStatsConcurrency)
	}
	if maxSeen == 0 {
		t.Error("fetch was never called")
	}
}

// The fan-out must also bound the number of goroutines alive at once, not just
// the number of concurrent fetches: a host with hundreds of containers should
// never park one goroutine per container waiting on the concurrency gate. With
// the slot acquired before the goroutine is spawned, only the cap's worth stay
// alive while a slow fetch blocks.
func TestCollectDockerStatsBoundsSpawnedGoroutines(t *testing.T) {
	const n = 200
	containers := make([]shared.DockerContainer, n)
	for i := range containers {
		containers[i].ID = strconv.Itoa(i)
	}

	release := make(chan struct{})
	fetch := func(_ context.Context, _ string) (shared.DockerContainer, error) {
		<-release
		return shared.DockerContainer{CPUPercent: 1}, nil
	}

	base := runtime.NumGoroutine()
	done := make(chan struct{})
	go func() {
		collectDockerStats(context.Background(), containers, fetch)
		close(done)
	}()

	// Let the fan-out reach steady state against the blocked fetch.
	time.Sleep(100 * time.Millisecond)
	peak := runtime.NumGoroutine() - base

	close(release)
	<-done

	if peak > maxDockerStatsConcurrency*4 {
		t.Errorf("parked %d goroutines at once for %d containers; fan-out should stay near the cap %d", peak, n, maxDockerStatsConcurrency)
	}
}
