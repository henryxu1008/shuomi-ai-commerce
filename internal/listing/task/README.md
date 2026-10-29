# Listing Canonical Subject

This package retains `CanonicalSubjectReader`, `CanonicalSubject`, `Actor` and
pure validation/authorization rules for task-to-product reads. Callers provide a
verified server identity; reads stay inside its tenant and subject owner scope.
`app/listingrecordstore`, `app/reportcenter` and `listing/record` consume this
existing contract. The repository/fact owner and authorization behavior are unchanged.

It does not implement task lifecycle or workflow orchestration. The old preview
and unimplemented directory declarations are source-retired under the
[confirmed #301 boundary](../../../docs/architecture/legacy-listing-preview-download-closure-retirement-boundary.md);
`canonical_subject.go` and its tests remain intact.
