package eval

// Strata of demand, by kind: each priority, each modality, each priority
// and modality together (where the denominators allow), and the inherited
// orders. A stratum key is "kind:value".
const (
	KindPriority         = "priority"
	KindModality         = "modality"
	KindPriorityModality = "priorityModality"
	KindInherited        = "inherited"
)

var strataKinds = []string{KindPriority, KindModality, KindPriorityModality, KindInherited}

// strataOf are the keys of an order or study's strata.
func strataOf(priority, modality string) []string {
	return []string{KindPriority + ":" + priority, KindModality + ":" + modality, KindPriorityModality + ":" + priority + "/" + modality}
}

// InheritedStratum is the key of the inherited orders' stratum.
const InheritedStratum = KindInherited + ":all"

func kindOf(key string) string {
	for i := range len(key) {
		if key[i] == ':' {
			return key[:i]
		}
	}
	return key
}
