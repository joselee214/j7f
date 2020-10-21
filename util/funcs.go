package util



// ArrayColumn array_column()
func ArrayColumnStrings(input []map[string]string, columnKey string) []string {
	columns := make([]string, 0, len(input))
	for _, val := range input {
		if v, ok := val[columnKey]; ok {
			columns = append(columns, v)
		}
	}
	return columns
}

func Contains(slice []string, s string) int {
	for index, value := range slice {
		if value == s {
			return index
		}
	}
	return -1
}