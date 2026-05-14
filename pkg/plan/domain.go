package plan

import (
	"encoding/json"
	"os/exec"
)

func InferDomain(rosterDescription string, rosterLabels []string, explicitDomain string, cwd string) (string, string) {
	if explicitDomain != "" {
		return explicitDomain, "explicit"
	}

	brPath, err := exec.LookPath("lr")
	if err != nil {
		return "", "none"
	}

	domains := listLoreDomains(brPath, cwd)
	if len(domains) == 0 {
		return "", "none"
	}

	domainSet := make(map[string]bool)
	for _, d := range domains {
		domainSet[d] = true
	}

	for _, label := range rosterLabels {
		if domainSet[label] {
			return label, "labels"
		}
	}

	return "", "none"
}

func listLoreDomains(brPath, cwd string) []string {
	cmd := exec.Command(brPath, "--json", "status")
	cmd.Dir = cwd
	out, err := cmd.Output()
	if err != nil {
		return nil
	}

	var data struct {
		Domains []struct {
			Domain string `json:"domain"`
		} `json:"domains"`
	}
	if err := json.Unmarshal(out, &data); err != nil {
		return nil
	}

	var res []string
	for _, d := range data.Domains {
		res = append(res, d.Domain)
	}
	return res
}
