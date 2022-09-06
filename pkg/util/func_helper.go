package util

func InArray(s string, arr []string) bool {
	for _, eachItem := range arr {
		if eachItem == s {
			return true
		}
	}
	return false
}

func Array_merge(arr ...[]string) []string {
	result := []string{}
	for _, arrItem := range arr {
		result = append(result, arrItem...)
	}
	return result
}
