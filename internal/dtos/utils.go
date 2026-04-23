package dtos

func DerefString(s *string, defaultValue string) string {
	if s == nil {
		return defaultValue
	}

	return *s
}

func DerefInt(i *int, defaultValue int) int {
	if i == nil {
		return defaultValue
	}

	return *i
}
