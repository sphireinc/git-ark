package backup

import "time"

const MetadataSchemaVersion = 1

// MetadataHistoryLimit keeps the metadata file useful without growing forever.
const MetadataHistoryLimit = 20

type RunOptions struct {
	RepoPath        string
	ModeOverride    string
	SelectedRemotes []string
	IncludeDisabled bool
	DryRun          bool
	Yes             bool
}

type TargetPlan struct {
	Name        string
	URL         string
	CurrentURL  string
	Enabled     bool
	Required    bool
	Selected    bool
	ExistsLocal bool
	UseURL      bool
	WillMutate  bool
	WillPush    bool
}

// SyncAction explains how a target remote will be synchronized.
func (t TargetPlan) SyncAction(manageRemotes bool) string {
	if !manageRemotes {
		if !t.ExistsLocal {
			return "push-direct"
		}
		return "push-local-remote"
	}
	if !t.ExistsLocal {
		return "add-remote"
	}
	if t.CurrentURL != "" && t.CurrentURL != t.URL {
		return "update-remote-url"
	}
	return "reuse-remote"
}

type Plan struct {
	RepoPath         string
	ResolvedRepoRoot string
	Mode             string
	Targets          []TargetPlan
	SelectedRemotes  []string
	RequiredRemotes  []string
	Branches         []string
	Tags             []string
	BranchPushMode   string
	TagPushMode      string
	PushAllRefs      bool
	PushNotes        bool
	BundleEnabled    bool
	BundleOnly       bool
	BundlePath       string
	RemoteManagement bool
	LFSNote          string
	DangerousOps     []string
	LFSDetected      bool
	SkipLFS          bool
	DryRun           bool
	WillFetch        bool
	WillPull         bool
}

type TargetResult struct {
	Target   string `json:"target"`
	Type     string `json:"type"`
	URL      string `json:"url"`
	Required bool   `json:"required"`
	Success  bool   `json:"success"`
	Error    string `json:"error"`
}

// Report is the on-disk summary of one backup run.
type Report struct {
	Repo       string         `json:"repo"`
	Mode       string         `json:"mode"`
	DryRun     bool           `json:"dry_run"`
	StartedAt  time.Time      `json:"started_at"`
	FinishedAt time.Time      `json:"finished_at"`
	DurationMS int64          `json:"duration_ms"`
	Results    []TargetResult `json:"results"`
	Success    bool           `json:"success"`
	Errors     []string       `json:"errors"`
}

// MetadataFile stores the recent backup history for a repo.
type MetadataFile struct {
	SchemaVersion int      `json:"schema_version"`
	Repo          string   `json:"repo"`
	History       []Report `json:"history"`
}

// LatestReport returns the newest backup entry if one exists.
func (m MetadataFile) LatestReport() (Report, bool) {
	if len(m.History) == 0 {
		return Report{}, false
	}
	return m.History[0], true
}

// RecentReports returns the most recent reports up to the requested limit.
func (m MetadataFile) RecentReports(limit int) []Report {
	if limit <= 0 || len(m.History) == 0 {
		return nil
	}
	if limit > len(m.History) {
		limit = len(m.History)
	}
	out := make([]Report, limit)
	copy(out, m.History[:limit])
	return out
}

// prepend keeps the newest run at the top and trims old history off the end.
func (m *MetadataFile) prepend(report Report) {
	if m == nil {
		return
	}
	m.History = append([]Report{report}, m.History...)
	if len(m.History) > MetadataHistoryLimit {
		m.History = m.History[:MetadataHistoryLimit]
	}
	if report.Repo != "" {
		m.Repo = report.Repo
	}
	if m.SchemaVersion == 0 {
		m.SchemaVersion = MetadataSchemaVersion
	}
}
