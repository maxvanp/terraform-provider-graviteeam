package reconcile

import "sort"

// DiffStrings returns the items to add and remove to reconcile current toward desired.
func DiffStrings(desired, current []string) (toAdd []string, toRemove []string) {
	desiredSet := make(map[string]struct{}, len(desired))
	for _, item := range desired {
		desiredSet[item] = struct{}{}
	}

	currentSet := make(map[string]struct{}, len(current))
	for _, item := range current {
		currentSet[item] = struct{}{}
	}

	for item := range desiredSet {
		if _, ok := currentSet[item]; !ok {
			toAdd = append(toAdd, item)
		}
	}
	for item := range currentSet {
		if _, ok := desiredSet[item]; !ok {
			toRemove = append(toRemove, item)
		}
	}

	sort.Strings(toAdd)
	sort.Strings(toRemove)
	return toAdd, toRemove
}
