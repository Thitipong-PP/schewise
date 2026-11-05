// SPDX-License-Identifier: Apache-2.0

// apigen emits the OpenAPI contract from the DTOs used by HTTP handlers.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"reflect"

	"github.com/Thitipong-PP/schewise/internal/apicontract"
)

func main() {
	check := flag.Bool("check", false, "Check the committed OpenAPI contract without writing")
	flag.Parse()
	data, err := apicontract.JSON()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	path := "docs/openapi.json"
	if *check {
		actual, err := os.ReadFile(path)
		if err != nil {
			fmt.Fprintln(os.Stderr, "OpenAPI contract missing; run make api-generate")
			os.Exit(1)
		}
		var want, got any
		if json.Unmarshal(data, &want) != nil || json.Unmarshal(actual, &got) != nil || !reflect.DeepEqual(want, got) {
			fmt.Fprintln(os.Stderr, "OpenAPI contract is stale; run make api-generate")
			os.Exit(1)
		}
		fmt.Println("OpenAPI contract matches backend DTOs and routes.")
		return
	}
	if err := os.WriteFile(path, append(data, '\n'), 0644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
