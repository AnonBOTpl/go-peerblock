package updater

import (
	"time"

	"go-peerblock/core"
)

// Source defines an IP blocklist source.
type Source struct {
	Name        string    `json:"name"`
	URL         string    `json:"url"`
	Format      int       `json:"format"` // core.Format value
	Enabled     bool      `json:"enabled"`
	APIKey      string    `json:"api_key"`
	Description string    `json:"description"`
	LastSync    time.Time `json:"last_sync"`
	RangeCount  int       `json:"range_count"`
}

// DefaultSources provides a sensible set of initial sources.
var DefaultSources = []Source{
	{
		Name:        "firehol-level1",
		URL:         "https://raw.githubusercontent.com/firehol/blocklist-ipsets/master/firehol_level1.netset",
		Format:      int(core.FormatCIDR),
		Enabled:     true,
		Description: "FireHOL Level 1 — a safe blocklist of attacks, malware, and known malicious IPs. Updated daily. Low false positive rate.",
	},
	{
		Name:        "firehol-level2",
		URL:         "https://raw.githubusercontent.com/firehol/blocklist-ipsets/master/firehol_level2.netset",
		Format:      int(core.FormatCIDR),
		Enabled:     true,
		Description: "FireHOL Level 2 — extended blocklist (includes level1 + additional sources). Higher block rate but may produce false positives.",
	},
	{
		Name:        "firehol-level3",
		URL:         "https://raw.githubusercontent.com/firehol/blocklist-ipsets/master/firehol_level3.netset",
		Format:      int(core.FormatCIDR),
		Enabled:     false,
		Description: "FireHOL Level 3 — the most aggressive blocklist. Blocks ~200k+ ranges. May block legitimate services — use with caution.",
	},
	{
		Name:        "spamhaus-drop",
		URL:         "https://www.spamhaus.org/drop/drop.txt",
		Format:      int(core.FormatCIDR),
		Enabled:     true,
		Description: "Spamhaus DROP (Do Not Route or Peer) — professional botnet, bulletproof hosting, and hijacked infrastructure blocklist. Continuously updated.",
	},
	{
		Name:        "spamhaus-edrop",
		URL:         "https://www.spamhaus.org/drop/edrop.txt",
		Format:      int(core.FormatCIDR),
		Enabled:     false,
		Description: "Spamhaus EDROP — extended DROP list. Additional IP ranges beyond the main list.",
	},
	{
		Name:        "emerging-threats",
		URL:         "https://rules.emergingthreats.net/fwrules/emerging-Block-IPs.txt",
		Format:      int(core.FormatCIDR),
		Enabled:     true,
		Description: "Emerging Threats (Proofpoint) — C&C (Command & Control) server and botnet IPs. Updated every 6-12h. Contains individual IPs (/32).",
	},
	{
		Name:        "blocklist-de",
		URL:         "https://lists.blocklist.de/lists/all.txt",
		Format:      int(core.FormatCIDR),
		Enabled:     true,
		Description: "Blocklist.de — IPs that attacked servers in the last 48 hours. SSH, HTTP, FTP, Mail attacks. Updated every 1-2h. Contains individual IPs (/32).",
	},
	{
		Name:        "dshield",
		URL:         "https://www.dshield.org/block.txt",
		Format:      int(core.FormatCIDR),
		Enabled:     false,
		Description: "DShield (SANS) — top 20 attacking IPs globally according to DShield reports. Updated daily.",
	},
	{
		Name:        "alienvault",
		URL:         "https://reputation.alienvault.com/reputation.generic",
		Format:      int(core.FormatCIDR),
		Enabled:     false,
		Description: "AlienVault OTX — reputation list from AlienVault Open Threat Exchange. Malicious IPs from various threat categories.",
	},
	{
		Name:        "greensnow",
		URL:         "https://blocklist.greensnow.co/greensnow.txt",
		Format:      int(core.FormatCIDR),
		Enabled:     false,
		Description: "GreenSnow — malicious IPs detected by GreenSnow honeypots. Updated every few hours.",
	},
}
