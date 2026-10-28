// Package canonical defines the pure value contract used by Catalog projection
// and current SHEIN draft and offline-package consumers.
//
// These values preserve their existing JSON and source-trace fields. Source
// normalization and publication belong to product/sourcing and product/catalog;
// this package does not infer confidence, approve assets, or own runtime state.
package canonical
