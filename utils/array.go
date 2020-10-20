package utils

func IncludesString(strs []string, item string) bool {
	for _, str := range strs {
		if str == item {
			return true
		}
	}
	return false
}
