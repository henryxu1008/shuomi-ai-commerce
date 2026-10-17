import type { Ref } from "react"
import { useMergeRefs } from "@floating-ui/react"

// Reuse the existing Floating UI ref composer, including React 19 cleanup.
export function useComposedRef<T>(...refs: (Ref<T> | undefined)[]) {
  return useMergeRefs(refs)
}
