package utils

// ContainsString determines if a string is in an array of strings.
func ContainsString(list []string, str string) bool {
	if len(list) == 0 {
		return false
	}

	for item := range list {
		if str == list[item] {
			return true
		}
	}

	return false
}

// UniqueStrings gets the unique strings in a list of strings.
func UniqueStrings(list []string) []string {
	uniqueMap := make(map[string]bool)

	uniqueList := []string{}

	for item := range list {
		if !uniqueMap[list[item]] {
			uniqueList = append(uniqueList, list[item])

			uniqueMap[list[item]] = true
		}
	}

	return uniqueList
}
