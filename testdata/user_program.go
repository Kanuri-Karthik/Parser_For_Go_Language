package main

import "fmt"

func main() {
	var n int

	fmt.Print("Enter a number: ")
	fmt.Scan(&n)

	if n%2 == 0 {
		fmt.Println(n, "is even")
	} else {
		fmt.Println(n, "is odd")
	}

	sum := 0
	for i := 1; i <= n; i++ {
		sum += i
	}

	fmt.Println("Sum:", sum)
}
