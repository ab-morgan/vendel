import { createRootRoute, HeadContent, Outlet } from "@tanstack/react-router"
import { lazy, Suspense } from "react"
import ErrorComponent from "@/components/Common/ErrorComponent"
import NotFound from "@/components/Common/NotFound"

// Lazy-load devtools only in development. In production import.meta.env.DEV is
// false (compile-time constant), so both branches resolve to () => null and
// Vite tree-shakes the devtools packages out of the production bundle.
const TanStackRouterDevtools = import.meta.env.DEV
  ? lazy(() =>
      import("@tanstack/react-router-devtools").then((m) => ({
        default: m.TanStackRouterDevtools,
      }))
    )
  : () => null

const ReactQueryDevtools = import.meta.env.DEV
  ? lazy(() =>
      import("@tanstack/react-query-devtools").then((m) => ({
        default: m.ReactQueryDevtools,
      }))
    )
  : () => null

export const Route = createRootRoute({
  component: () => (
    <>
      <HeadContent />
      <Outlet />
      <Suspense>
        <TanStackRouterDevtools position="bottom-right" />
      </Suspense>
      <Suspense>
        <ReactQueryDevtools initialIsOpen={false} />
      </Suspense>
    </>
  ),
  notFoundComponent: () => <NotFound />,
  errorComponent: () => <ErrorComponent />,
})
