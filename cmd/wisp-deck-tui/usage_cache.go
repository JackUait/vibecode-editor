package main

import (
	"os"
	"path/filepath"
	"time"

	"github.com/jackuait/wisp-deck/internal/subusage"
)

// refreshUsageCache runs one throttled, single-flight fetch into a usage cache
// file. fetch is called only once both gates pass, so a skipped round costs no
// network round trip and no Keychain read.
//
// A failed fetch keeps the last good reading and advances only checked_at.
// fetched_at is the freshness gate every reader displays from, so blanking it
// would drop a working subscription's numbers the first time its endpoint
// hiccups — and for the All-In picker that also means dropping its rows.
func refreshUsageCache(cache string, minInterval int, fetch func() (subusage.Snapshot, error)) {
	if cache == "" {
		return
	}
	prev, prevErr := subusage.ReadCache(cache)
	if prevErr == nil && time.Now().Unix()-prev.CheckedAt < int64(minInterval) {
		return
	}
	// The cache's directory must exist before the lock beside it can.
	if err := os.MkdirAll(filepath.Dir(cache), 0o700); err != nil {
		return
	}
	// Single-flight across concurrent launches and statusline ticks; a lock
	// older than two minutes belongs to a run that died holding it.
	lock := cache + ".lock"
	if info, err := os.Stat(lock); err == nil {
		if time.Since(info.ModTime()) < 2*time.Minute {
			return
		}
		_ = os.Remove(lock)
	}
	f, err := os.OpenFile(lock, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return
	}
	_ = f.Close()
	defer func() { _ = os.Remove(lock) }()

	snap, fetchErr := fetch()
	snap.CheckedAt = time.Now().Unix()
	if fetchErr != nil {
		if prevErr == nil {
			snap.RateLimits = prev.RateLimits
			snap.FetchedAt = prev.FetchedAt
		}
	} else {
		snap.FetchedAt = snap.CheckedAt
	}
	_ = subusage.WriteCache(cache, snap)
}
