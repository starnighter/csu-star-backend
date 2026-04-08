package service

func externalUserRole(role string) string {
	switch role {
	case "user":
		return "student"
	default:
		return role
	}
}

func intPtr(v int) *int {
	return &v
}

func int64Ptr(v int64) *int64 {
	return &v
}

func valueOrZero(v *int) int {
	if v == nil {
		return 0
	}
	return *v
}
