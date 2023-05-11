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
