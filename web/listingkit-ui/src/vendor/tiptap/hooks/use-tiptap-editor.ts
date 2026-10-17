import * as React from "react"
import type { Editor } from "@tiptap/react"
import { useCurrentEditor, useEditorState } from "@tiptap/react"

/**
 * Hook that provides access to a Tiptap editor instance.
 *
 * Accepts an optional editor instance directly, or falls back to retrieving
 * the editor from the Tiptap context if available. This allows components
 * to work both when given an editor directly and when used within a Tiptap
 * editor context.
 *
 * @param providedEditor - Optional editor instance to use instead of the context editor
 * @returns The provided editor or the editor from context, whichever is available
 */
export function useTiptapEditor(providedEditor?: Editor | null): {
  editor: Editor | null
  editorState?: Editor["state"]
  canCommand?: Editor["can"]
} {
  const { editor: coreEditor } = useCurrentEditor()
  const mainEditor = React.useMemo(
    () => providedEditor || coreEditor,
    [providedEditor, coreEditor]
  )

  // Subscribe to a primitive transaction identity. Editor/state objects are
  // mutable; returning a prior selector object can leave controls on null after
  // deferred editor creation or miss stored-mark-only transactions.
  useEditorState({
    editor: mainEditor,
    selector: ({ transactionNumber }) => transactionNumber,
  })

  return {
    editor: mainEditor,
    editorState: mainEditor?.state,
    canCommand: mainEditor?.can,
  }
}
