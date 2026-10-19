// Package shein assembles SHEIN draft packages from canonical
// products and processed image assets.
//
// This package owns the current canonical-to-SHEIN draft/value contract, local
// assembly, offline validation preparation, and strict package codecs.
// canonical.Product is the platform-neutral product fact input, Package is the
// SHEIN draft/value container, and DraftPayload is the payload for current writes.
//
// Low-level SHEIN API calls stay in internal/shein/api. Official submission
// intents, receipts, and execution coordination belong to internal/listing/submission
// and its current application and persistence adapters.
package shein
