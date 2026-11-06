package mcp

import _ "embed"

// clusterAPILogo is the Cluster API logo the server advertises as its MCP icon.
// Source: github.com/kubernetes-sigs/cluster-api docs/logos/kubernetes-cluster-logos_final-02.svg
// at ceb414c6e24886ef868eb95b07cfd0840dac3c43, minified losslessly with styles inlined.
//
//go:embed cluster-api.svg
var clusterAPILogo []byte
