// Package simtest provides the public testing layer above the deterministic
// simulator.
//
// Harnesses record externally visible actions with Probe, run simulations with
// Execute, and attach safety and liveness checkers that convert traces into
// stable JSON reports for the platform UI.
package simtest
