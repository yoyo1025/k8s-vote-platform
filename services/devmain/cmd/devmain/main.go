package main

import (
	"fmt"

	"github.com/yoyo1025/k8s-vote-platform/libs/authjwt"
)

func main() {
	fmt.Println("devmain:", authjwt.Stub())
}
