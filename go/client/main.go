package main

import (
	_ "embed"
	"github.com/traefik/yaegi/interp"
	"github.com/traefik/yaegi/stdlib"
)

//go:embed client.go
var program string

func main() {

	i := interp.New(interp.Options{})
	err := i.Use(stdlib.Symbols)
	if err != nil {
		panic(err)
	}

	program += program + "run()"

	_, err = i.Eval(program)

}
