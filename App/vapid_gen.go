//go:build ignore

package main

import (
	"fmt"

	"github.com/SherClockHolmes/webpush-go"
)

func main() {
	vapidPrivate, vapidPublic, err := webpush.GenerateVAPIDKeys()
	if err != nil {
		panic(err)
	}

	fmt.Println("VAPIDPublicKey = \"" + vapidPublic + "\"")
	fmt.Println("VAPIDPrivateKey = \"" + vapidPrivate + "\"")
}
