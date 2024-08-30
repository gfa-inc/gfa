package ptr

func ToInt(v int) *int {
	return &v
}

func ToInt64(v int64) *int64 {
	return &v
}

func ToString(v string) *string {
	return &v
}

func ToBool(v bool) *bool {
	return &v
}
