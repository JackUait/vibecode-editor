// Package allin routes one Claude Code session across every configured
// subscription: each picker row names its source in the model id, and the
// loopback router swaps the credential for that source.
package allin

import "strings"

// Kind says which credential a picker row asks for.
type Kind int

const (
	// KindSession leaves the session's own credential and endpoint alone.
	KindSession Kind = iota
	KindAccount
	KindConfig
)

// Target is one parsed picker row.
type Target struct {
	Kind   Kind
	Source string
	Model  string
	Want1M bool
}

const (
	rowPrefix     = "wisp/"
	accountPrefix = "acct."
	configPrefix  = "cfg."
)

// Route parses a picker row. A row this build does not recognise is the
// session's own, never an error: the turn must still run.
func Route(model string) Target {
	trimmed, want1m := strip1M(model)
	rest, ok := strings.CutPrefix(trimmed, rowPrefix)
	if !ok {
		return Target{Kind: KindSession, Model: trimmed, Want1M: want1m}
	}
	source, id, ok := strings.Cut(rest, "/")
	if !ok || source == "" || id == "" {
		return Target{Kind: KindSession, Model: trimmed, Want1M: want1m}
	}
	switch {
	case strings.HasPrefix(source, accountPrefix):
		return Target{Kind: KindAccount, Source: source[len(accountPrefix):], Model: id, Want1M: want1m}
	case strings.HasPrefix(source, configPrefix):
		return Target{Kind: KindConfig, Source: source[len(configPrefix):], Model: id, Want1M: want1m}
	}
	return Target{Kind: KindSession, Model: trimmed, Want1M: want1m}
}

// strip1M removes the marker Claude Code reads off the raw model string to
// grant a 1M window. The upstream never sees it; the beta header carries it.
func strip1M(model string) (string, bool) {
	if len(model) >= 4 && strings.EqualFold(model[len(model)-4:], "[1m]") {
		return model[:len(model)-4], true
	}
	return model, false
}
