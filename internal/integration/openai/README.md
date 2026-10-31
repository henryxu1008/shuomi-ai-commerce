# OpenAI-compatible source image integration

Owns the current source-image edit adapter and the minimal organization-scoped
credential configuration used by published routes. Current text/Eino execution
continues to use its existing owner and exact credential lookup.

The old SDK Client, Manager, request pool, generic Subject/Scene/Reviewer adapters,
member-first fallback and global `openai` configuration were retired in #1033.
`SourceImageAdapter` performs the existing source edit and conservative quote;
it does not reconstruct the retired generic image pipeline.

Physical credential rows and `CredentialReference`/`CredentialVersion` are
unchanged. `ResolveClientConfig(ctx, name)` only reads the verified organization's
exact row; missing or disabled credentials fail closed without fallback.
