// Package resilience provides runtime-neutral retry primitives built on
// mature third-party libraries.
//
// The package is shared by domains and application code that need retry and
// backoff behavior without depending on runtime infrastructure.
// Implementations are backed by github.com/cenkalti/backoff/v5.
package resilience
