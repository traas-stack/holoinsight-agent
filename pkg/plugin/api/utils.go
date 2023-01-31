package api

func BoolToFloat64(b bool) float64 {
	if b {
		return 1
	} else {
		return 0
	}
}
