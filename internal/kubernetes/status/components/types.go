package components

import "k8s.io/api/core/v1"

// MatchData is a collection of data used to determine if a component is present
// in the cluster
type MatchData struct {
	Pods []v1.Pod
}

const (
	missingComponentString = "componentMissing"
	unknownVersionString   = "unknownVersion"
)
