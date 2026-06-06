package updater

import "time"

// Source defines an IP blocklist source.
type Source struct {
	Name        string    `json:"name"`
	URL         string    `json:"url"`
	Format      int       `json:"format"` // core.Format value
	Enabled     bool      `json:"enabled"`
	APIKey      string    `json:"api_key"`
	Description string    `json:"description"`
	LastSync    time.Time `json:"last_sync"`
}

// DefaultSources provides a sensible set of initial sources.
var DefaultSources = []Source{
	{
		Name:        "firehol-level1",
		URL:         "https://raw.githubusercontent.com/firehol/blocklist-ipsets/master/firehol_level1.netset",
		Format:      3, // FormatCIDR
		Enabled:     true,
		Description: "FireHOL Level 1 — bezpieczna lista blokad ataków, malware i znanych złośliwych IP. Aktualizowana codziennie. Niski wskaźnik fałszywych trafień.",
	},
	{
		Name:        "firehol-level2",
		URL:         "https://raw.githubusercontent.com/firehol/blocklist-ipsets/master/firehol_level2.netset",
		Format:      3, // FormatCIDR
		Enabled:     true,
		Description: "FireHOL Level 2 — rozszerzona lista blokad (zawiera level1 + dodatkowe źródła). Wyższy wskaźnik blokad, ale możliwe fałszywe trafienia.",
	},
	{
		Name:        "firehol-level3",
		URL:         "https://raw.githubusercontent.com/firehol/blocklist-ipsets/master/firehol_level3.netset",
		Format:      3, // FormatCIDR
		Enabled:     false,
		Description: "FireHOL Level 3 — najbardziej agresywna lista blokad. Blokuje ~200k+ zakresów. Może blokować legalne usługi — używaj ostrożnie.",
	},
	{
		Name:        "spamhaus-drop",
		URL:         "https://www.spamhaus.org/drop/drop.txt",
		Format:      3, // FormatCIDR
		Enabled:     true,
		Description: "Spamhaus DROP (Do Not Route or Peer) — profesjonalna lista blokad botnetów, bulletproof hostingu i przejętej infrastruktury. Aktualizowana na bieżąco.",
	},
	{
		Name:        "spamhaus-edrop",
		URL:         "https://www.spamhaus.org/drop/edrop.txt",
		Format:      3, // FormatCIDR
		Enabled:     false,
		Description: "Spamhaus EDROP — rozszerzona lista DROP. Dodatkowe zakresy IP poza główną listą.",
	},
	{
		Name:        "emerging-threats",
		URL:         "https://rules.emergingthreats.net/fwrules/emerging-Block-IPs.txt",
		Format:      3, // FormatCIDR
		Enabled:     true,
		Description: "Emerging Threats (Proofpoint) — lista IP serwerów C&C (Command & Control) i botnetów. Aktualizowana co 6-12h. Zawiera pojedyncze IP (/32).",
	},
	{
		Name:        "blocklist-de",
		URL:         "https://lists.blocklist.de/lists/all.txt",
		Format:      3, // FormatCIDR
		Enabled:     true,
		Description: "Blocklist.de — lista IP które atakowały serwery w ciągu ostatnich 48h. SSH, HTTP, FTP, Mail ataki. Aktualizowana co 1-2h. Zawiera pojedyncze IP (/32).",
	},
	{
		Name:        "dshield",
		URL:         "https://www.dshield.org/block.txt",
		Format:      3, // FormatCIDR
		Enabled:     false,
		Description: "DShield (SANS) — top 20 atakujących IP w skali globalnej według raportów DShield. Aktualizowana codziennie.",
	},
	{
		Name:        "alienvault",
		URL:         "https://reputation.alienvault.com/reputation.generic",
		Format:      3, // FormatCIDR
		Enabled:     false,
		Description: "AlienVault OTX — reputation list od AlienVault Open Threat Exchange. Złośliwe IP z różnych kategorii zagrożeń.",
	},
	{
		Name:        "greensnow",
		URL:         "https://blocklist.greensnow.co/greensnow.txt",
		Format:      3, // FormatCIDR
		Enabled:     false,
		Description: "GreenSnow — lista złośliwych IP wykrytych przez honeypoty GreenSnow. Aktualizowana co kilka godzin.",
	},
}
