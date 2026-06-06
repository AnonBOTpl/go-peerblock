package updater

import "time"

// Source defines an IP blocklist source.
type Source struct {
	Name     string    `json:"name"`
	URL      string    `json:"url"`
	Format   int       `json:"format"` // core.Format value
	Enabled  bool      `json:"enabled"`
	LastSync time.Time `json:"last_sync"`
}

// DefaultSources provides a sensible set of initial sources.
var DefaultSources = []Source{
	{
		Name:    "firehol-level1",
		URL:     "https://raw.githubusercontent.com/firehol/blocklist-ipsets/master/firehol_level1.netset",
		Format:  3, // FormatCIDR
		Enabled: true,
	},
	{
		Name:    "firehol-level2",
		URL:     "https://raw.githubusercontent.com/firehol/blocklist-ipsets/master/firehol_level2.netset",
		Format:  3, // FormatCIDR
		Enabled: true,
	},
	{
		Name:    "spamhaus-drop",
		URL:     "https://www.spamhaus.org/drop/drop.txt",
		Format:  3, // FormatCIDR
		Enabled: true,
	},
}
