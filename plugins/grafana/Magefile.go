//go:build mage

package main

import (
	// mage:import
	build "github.com/grafana/grafana-plugin-sdk-go/build"
)

// Default builds the platforms supported by the Grafana SDK.
var Default = build.BuildAll
