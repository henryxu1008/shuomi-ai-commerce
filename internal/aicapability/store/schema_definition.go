package store

// InvocationSchemaModel exposes the current owner's model solely for native
// read-only schema qualification. It is not a second persistence model.
func InvocationSchemaModel() any { return &invocationRow{} }
