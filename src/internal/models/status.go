package models

const (
	IdeaStatusInbox    = "inbox"
	IdeaStatusTriaged  = "triaged"
	IdeaStatusPromoted = "promoted"
	IdeaStatusRejected = "rejected"
)

var ValidIdeaStatuses = []string{IdeaStatusInbox, IdeaStatusTriaged, IdeaStatusPromoted, IdeaStatusRejected}

const (
	BugStatusOpen       = "open"
	BugStatusTriaged    = "triaged"
	BugStatusInProgress = "in-progress"
	BugStatusFixed      = "fixed"
	BugStatusWontFix    = "wont-fix"
)

var ValidBugStatuses = []string{BugStatusOpen, BugStatusTriaged, BugStatusInProgress, BugStatusFixed, BugStatusWontFix}

const (
	FeatureStatusProposed   = "proposed"
	FeatureStatusTriaged    = "triaged"
	FeatureStatusInProgress = "in-progress"
	FeatureStatusDone       = "done"
	FeatureStatusRejected   = "rejected"
)

var ValidFeatureStatuses = []string{FeatureStatusProposed, FeatureStatusTriaged, FeatureStatusInProgress, FeatureStatusDone, FeatureStatusRejected}

const (
	PriorityLow    = "low"
	PriorityMedium = "medium"
	PriorityHigh   = "high"
)

const (
	CriticalityLow      = "low"
	CriticalityMedium   = "medium"
	CriticalityHigh     = "high"
	CriticalityCritical = "critical"
)

const (
	LevelLow    = "low"
	LevelMedium = "medium"
	LevelHigh   = "high"
)

func isIn(s string, valid []string) bool {
	for _, v := range valid {
		if v == s {
			return true
		}
	}
	return false
}

func IsValidIdeaStatus(s string) bool    { return isIn(s, ValidIdeaStatuses) }
func IsValidBugStatus(s string) bool     { return isIn(s, ValidBugStatuses) }
func IsValidFeatureStatus(s string) bool { return isIn(s, ValidFeatureStatuses) }
func IsValidPriority(s string) bool {
	return isIn(s, []string{PriorityLow, PriorityMedium, PriorityHigh})
}
func IsValidCriticality(s string) bool {
	return isIn(s, []string{CriticalityLow, CriticalityMedium, CriticalityHigh, CriticalityCritical})
}
func IsValidLevel(s string) bool { return isIn(s, []string{LevelLow, LevelMedium, LevelHigh}) }
