package api

import (
	"archive/zip"
	"bytes"
	"fmt"
	"strings"
	"time"

	"github.com/hiro1202/fourkeys-insights/internal/db"
	"github.com/hiro1202/fourkeys-insights/internal/metrics"
)

// UTF-8 BOM for Excel compatibility
var bom = []byte{0xEF, 0xBB, 0xBF}

func buildExportZIP(group *db.Group, prs []*db.PullRequest, result metrics.FourKeysResult, rulesMap map[int64]metrics.IncidentRules, leadTimeStart string) ([]byte, error) {
	buf := new(bytes.Buffer)
	w := zip.NewWriter(buf)

	// 1. metrics_summary.csv
	summary, err := w.Create("metrics_summary.csv")
	if err != nil {
		return nil, err
	}
	summary.Write(bom)

	now := time.Now().UTC()
	period := metrics.CalcPeriod(now, group.AggregationUnit)

	// Build incident rules snapshot for header
	rules := metrics.ParseIncidentRules(group.IncidentRules)
	rulesSnapshot := fmt.Sprintf("title_keywords=%s; branch_keywords=%s; labels=%s",
		strings.Join(rules.TitleKeywords, ","),
		strings.Join(rules.BranchKeywords, ","),
		strings.Join(rules.Labels, ","))

	mttrStr := "N/A"
	if result.MTTRHours != nil {
		mttrStr = fmt.Sprintf("%.1f", *result.MTTRHours)
	}

	fmt.Fprintf(summary, "# Exported: %s\n", now.Format(time.RFC3339))
	fmt.Fprintf(summary, "# Group: %s\n", group.Name)
	fmt.Fprintf(summary, "# Aggregation: %s\n", group.AggregationUnit)
	fmt.Fprintf(summary, "# Lead Time Start: %s\n", leadTimeStart)
	fmt.Fprintf(summary, "# Incident Rules: %s\n", rulesSnapshot)
	fmt.Fprintf(summary, "Period,Lead Time (hours),Deployment Frequency (/day),Change Failure Rate (%%),MTTR (hours),DORA Level\n")
	fmt.Fprintf(summary, "%s - %s,%.1f,%.2f,%.1f,%s,%s\n",
		period.Start.Format("2006-01-02"), period.End.Format("2006-01-02"),
		result.LeadTimeHours, result.DeployFrequency, result.ChangeFailureRate,
		mttrStr, result.OverallLevel)

	// 2. pull_requests.csv
	prFile, err := w.Create("pull_requests.csv")
	if err != nil {
		return nil, err
	}
	prFile.Write(bom)
	fmt.Fprintf(prFile, "PR Number,Title,Repository,Branch,Merged At,Lead Time (hours),Is Incident,Linked Issue\n")

	for _, pr := range prs {
		lt := metrics.CalculateLeadTime(pr, leadTimeStart)

		rules, ok := rulesMap[pr.RepoID]
		if !ok {
			rules = metrics.DefaultIncidentRules()
		}
		isIncident := "No"
		if metrics.IsIncident(pr, rules) {
			isIncident = "Yes"
		}

		linkedIssue := ""
		if pr.LinkedIssueNumber.Valid {
			linkedIssue = fmt.Sprintf("#%d", pr.LinkedIssueNumber.Int64)
		}

		branch := ""
		if pr.BranchName.Valid {
			branch = pr.BranchName.String
		}

		// Escape CSV fields that might contain commas
		title := strings.ReplaceAll(pr.Title, "\"", "\"\"")
		if strings.ContainsAny(pr.Title, ",\"\n") {
			title = "\"" + title + "\""
		}

		fmt.Fprintf(prFile, "%d,%s,%s,%s,%s,%.1f,%s,%s\n",
			pr.PRNumber, title, pr.RepoFullName, branch,
			pr.MergedAt.Format(time.RFC3339),
			lt.LeadTime.Hours(), isIncident, linkedIssue)
	}

	if err := w.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
