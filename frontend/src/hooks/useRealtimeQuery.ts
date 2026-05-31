import { useQueryClient } from "@tanstack/react-query"
import { useEffect, useRef } from "react"
import pb from "@/lib/pocketbase"

export function useRealtimeQuery(collection: string, queryKeys: string[][]) {
  const queryClient = useQueryClient()
  const queryKeysRef = useRef(queryKeys)
  queryKeysRef.current = queryKeys

  useEffect(() => {
    if (!pb.authStore.isValid) return

    let cancelled = false
    let unsubFn: (() => Promise<void>) | null = null

    pb.collection(collection)
      .subscribe("*", () => {
        for (const key of queryKeysRef.current) {
          queryClient.invalidateQueries({ queryKey: key })
        }
      })
      .then((fn) => {
        if (cancelled) {
          // Component unmounted before subscribe resolved; release immediately.
          fn()
        } else {
          unsubFn = fn
        }
      })

    return () => {
      cancelled = true
      unsubFn?.()
    }
  }, [collection, queryClient])
}
