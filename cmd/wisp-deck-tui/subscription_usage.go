package main

import (
	"net/http"
	"time"

	"github.com/jackuait/wisp-deck/internal/subusage"
	"github.com/spf13/cobra"
)

// subscription-usage refreshes the on-disk 5h/7d usage snapshot for one
// subscription config. The statusline spawns it disowned on every render and
// reads only the cache file, so this command carries the throttle itself:
// a recent checked_at (success OR failure) exits without touching the
// network, and a lock file keeps concurrent statusline ticks single-flight.
// It never prints and never fails the caller — a statusline is no place for
// error output.
var (
	subUsageConfigsDir  string
	subUsageList        string
	subUsageConfig      string
	subUsageCache       string
	subUsageCodexAuth   string
	subUsageMinInterval int
)

var subscriptionUsageCmd = &cobra.Command{
	Use:           "subscription-usage",
	Short:         "Refresh the cached 5h/7d usage snapshot for a subscription config",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE:          runSubscriptionUsage,
}

func init() {
	subscriptionUsageCmd.Flags().StringVar(&subUsageConfigsDir, "configs-dir", "", "Directory holding the subscription settings JSONs")
	subscriptionUsageCmd.Flags().StringVar(&subUsageList, "list", "", "claude-configs.list path")
	subscriptionUsageCmd.Flags().StringVar(&subUsageConfig, "config", "", "Subscription config filename (e.g. zhipu-glm.json)")
	subscriptionUsageCmd.Flags().StringVar(&subUsageCache, "cache", "", "Snapshot cache file to maintain")
	subscriptionUsageCmd.Flags().StringVar(&subUsageCodexAuth, "codex-auth", "", "Override ~/.codex/auth.json (tests)")
	subscriptionUsageCmd.Flags().IntVar(&subUsageMinInterval, "min-interval", 300, "Seconds between refresh attempts")
	rootCmd.AddCommand(subscriptionUsageCmd)
}

func runSubscriptionUsage(cmd *cobra.Command, args []string) error {
	if subUsageConfig == "" || subUsageCache == "" || subUsageConfigsDir == "" {
		return nil
	}
	client := &http.Client{Timeout: 15 * time.Second}
	refreshUsageCache(subUsageCache, subUsageMinInterval, func() (subusage.Snapshot, error) {
		snap, _, err := subusage.Fetch(client, subUsageConfigsDir, subUsageList, subUsageConfig, subUsageCodexAuth)
		return snap, err
	})
	return nil
}
