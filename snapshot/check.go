package snapshot

import "fmt"

func check(err error) {
	if err != nil {
		fmt.Println(err)
	}
}

func checkFatal(err error) {
	if err != nil {
		panic(err)
	}
}
