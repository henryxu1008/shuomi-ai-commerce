import { useCallback, useSyncExternalStore } from "react"

export function useIsMobile(breakpoint = 768) {
  const query = `(max-width: ${breakpoint - 1}px)`
  const subscribe = useCallback((notify: () => void) => {
    const media = window.matchMedia(query)
    media.addEventListener("change", notify)
    return () => media.removeEventListener("change", notify)
  }, [query])
  const snapshot = useCallback(() => window.matchMedia(query).matches, [query])
  return useSyncExternalStore(subscribe, snapshot, () => false)
}
