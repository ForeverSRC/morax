package utils

func If(condition bool, t interface{}, f interface{}) interface{} {
	if condition {
		return t
	} else {
		return f
	}
}
