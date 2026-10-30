package main

var limit int = 10

func calc(a, b int) int {
	x := a + b*2
	x = x - (limit / 2)
	if x > 5 {
		return x
	}
	return -x
}
