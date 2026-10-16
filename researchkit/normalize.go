package researchkit

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"sort"
	"strings"
)

type Observation struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	Title    string `json:"title"`
	Severity string `json:"severity"`
	Source   string `json:"source"`
	Template string `json:"template,omitempty"`
	Status   string `json:"status"`
}
type Normalized struct {
	Source     string        `json:"source"`
	InputCount int           `json:"input_count"`
	Duplicates int           `json:"duplicates"`
	Total      int           `json:"total"`
	Items      []Observation `json:"items"`
	Truncated  bool          `json:"truncated"`
	Notice     string        `json:"notice"`
}

func Normalize(source, input string, limit int) (Normalized, error) {
	if err := CheckInput(input); err != nil {
		return Normalized{}, err
	}
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	raw := []Observation{}
	switch source {
	case "nuclei":
		var events []map[string]any
		if strings.HasPrefix(strings.TrimSpace(input), "[") {
			if err := json.Unmarshal([]byte(input), &events); err != nil {
				return Normalized{}, errors.New("invalid Nuclei JSON array")
			}
		} else {
			scanner := bufio.NewScanner(strings.NewReader(input))
			scanner.Buffer(make([]byte, 4096), MaxInput)
			line := 0
			for scanner.Scan() {
				line++
				if strings.TrimSpace(scanner.Text()) == "" {
					continue
				}
				var event map[string]any
				if json.Unmarshal(scanner.Bytes(), &event) != nil {
					return Normalized{}, fmt.Errorf("invalid Nuclei JSON at line %d", line)
				}
				events = append(events, event)
			}
			if scanner.Err() != nil {
				return Normalized{}, errors.New("Nuclei record exceeds input limit")
			}
		}
		for _, event := range events {
			info := obj(event["info"])
			location := str(event["matched-at"])
			if location == "" {
				location = str(event["host"])
			}
			title := str(info["name"])
			if title == "" {
				title = str(event["template-id"])
			}
			if location == "" || title == "" {
				return Normalized{}, errors.New("Nuclei event requires matched-at/host and info.name/template-id")
			}
			severity := strings.ToLower(str(info["severity"]))
			switch severity {
			case "critical", "high", "medium", "low", "info":
			default:
				severity = "info"
			}
			raw = append(raw, Observation{URL: cleanURL(location), Title: title, Severity: severity, Template: str(event["template-id"]), Source: source, Status: "unverified"})
		}
	case "ffuf":
		var doc struct {
			Results []struct {
				URL    string `json:"url"`
				Status int    `json:"status"`
			} `json:"results"`
		}
		if json.Unmarshal([]byte(input), &doc) != nil {
			return Normalized{}, errors.New("invalid ffuf JSON")
		}
		if doc.Results == nil {
			return Normalized{}, errors.New("ffuf input requires a results array")
		}
		for _, result := range doc.Results {
			if result.URL == "" {
				return Normalized{}, errors.New("ffuf result is missing its URL")
			}
			raw = append(raw, Observation{URL: cleanURL(result.URL), Title: fmt.Sprintf("HTTP resource returned %d", result.Status), Severity: "info", Source: source, Status: "unverified"})
		}
	case "nmap":
		var doc struct {
			XMLName xml.Name `xml:"nmaprun"`
			Hosts   []struct {
				Addresses []struct {
					Addr string `xml:"addr,attr"`
					Type string `xml:"addrtype,attr"`
				} `xml:"address"`
				Ports []struct {
					ID       string `xml:"portid,attr"`
					Protocol string `xml:"protocol,attr"`
					State    struct {
						Value string `xml:"state,attr"`
					} `xml:"state"`
					Service struct {
						Name string `xml:"name,attr"`
					} `xml:"service"`
				} `xml:"ports>port"`
			} `xml:"host"`
		}
		if xml.Unmarshal([]byte(input), &doc) != nil {
			return Normalized{}, errors.New("invalid Nmap XML")
		}
		for _, host := range doc.Hosts {
			address := ""
			for _, a := range host.Addresses {
				if a.Type == "ipv4" || a.Type == "ipv6" {
					address = a.Addr
					break
				}
			}
			if address == "" {
				continue
			}
			for _, port := range host.Ports {
				if port.State.Value != "open" {
					continue
				}
				raw = append(raw, Observation{URL: address + ":" + port.ID, Title: fmt.Sprintf("Open %s/%s: %s", port.Protocol, port.ID, port.Service.Name), Severity: "info", Source: source, Status: "unverified"})
			}
		}
	default:
		return Normalized{}, errors.New("source must be nuclei, ffuf or nmap")
	}
	seen := map[string]bool{}
	uniqueItems := []Observation{}
	for _, item := range raw {
		digest := sha256.Sum256([]byte(item.Source + "\x00" + item.URL + "\x00" + item.Title + "\x00" + item.Template))
		item.ID = hex.EncodeToString(digest[:])[:20]
		if seen[item.ID] {
			continue
		}
		seen[item.ID] = true
		uniqueItems = append(uniqueItems, item)
	}
	rank := map[string]int{"critical": 4, "high": 3, "medium": 2, "low": 1, "info": 0}
	sort.SliceStable(uniqueItems, func(i, j int) bool { return rank[uniqueItems[i].Severity] > rank[uniqueItems[j].Severity] })
	total := len(uniqueItems)
	truncated := total > limit
	if truncated {
		uniqueItems = uniqueItems[:limit]
	}
	return Normalized{Source: source, InputCount: len(raw), Duplicates: len(raw) - total, Total: total, Items: uniqueItems, Truncated: truncated, Notice: "Observations only, not confirmed vulnerabilities. Supplied severity is retained for Nuclei; discovery output is informational. Query values are removed before deduplication, so query-only differences may collapse. Nothing is scanned or saved to findings."}, nil
}
