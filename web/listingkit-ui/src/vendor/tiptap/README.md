# Tiptap official Simple Editor components

Source: https://github.com/ueberdosis/tiptap-ui-components/tree/799929bea4804c73767562b69f8acc2acdb8ac86/apps/web/src
License: MIT, Copyright (c) 2025 Tiptap; full license in LICENSE.

Actual transitive source components used by the official Simple Editor: Toolbar, heading/list menus, mark/quote/undo buttons, LinkPopover, primitives, hooks, icons and styles. Product composition: components/workbench/description-editor.tsx, adapted from the upstream simple-editor template with the existing bounded schema.

Local adaptations:
- Import paths isolated here; keep Tiptap 3.32.0.
- The upstream SCSS is retained as source and precompiled locally into CSS with official Dart Sass 1.93.2; production imports CSS and adds no Sass compiler/lifecycle dependency. Regenerate each SCSS with `sass.compile(path, {style: "expanded"})`; product styles use description-editor.scss as source.
- Include the required upstream keyframes and prefix animation names with tiptap- so portal exit animations finish without global name collisions.
- Scope root theme variables to editor/portal controls; remove global typography/reset; prefix white/black variables. App tokens override in description-editor.scss.
- Exclude demo fonts/theme/upload/content and unsupported extensions. Remove unused demo upload and broad URL utilities.
- Editor subscription includes the Tiptap transaction counter so stored-mark-only changes update toolbar state (covered by a real editor regression).
- Adapt React 19 refs with Floating UI useMergeRefs and media-query subscription with useSyncExternalStore; derive query/active-link UI state without effect cascades. No lint/CI rules are disabled.
- Link UI uses existing HTTPS-only descriptionURL and read-only checks; Chinese accessible labels and heading/list labels.

Design: docs/architecture/rich-description-simple-editor-images-v1.md. Content, authorization, uploads and recovery remain in existing owners, outside this UI library.
