import { Link, useRouter } from "@tanstack/react-router";
import { AlertTriangleIcon, HomeIcon, RotateCwIcon } from "lucide-react";
import { Button } from "@/components/ui/button";

interface RouteErrorProps {
  error: Error;
  info?: { componentStack?: string };
  reset?: () => void;
}

/**
 * Full-width route error panel (P0-4). Registered as errorComponent on the
 * _authenticated layout route so every authenticated page recovers from a
 * failed loader instead of crashing to a blank screen.
 *
 * Recovery: "Try again" re-runs the loader via router.invalidate() — per
 * TanStack Router docs, reset() alone clears the boundary but leaves the
 * stale loader error, so invalidate() is the correct retry path.
 */
export function RouteError({ error }: RouteErrorProps) {
  const router = useRouter();

  if (import.meta.env.DEV) {
    console.error("[route-error] loader/query failed:", error);
  }

  return (
    <div
      role="alert"
      className="flex w-full min-h-[60vh] items-center justify-center p-6"
    >
      <div className="max-w-md space-y-4 text-center">
        <AlertTriangleIcon className="mx-auto h-10 w-10 text-destructive" />
        <h2 className="text-lg font-semibold">Something went wrong</h2>
        <p className="text-sm text-muted-foreground">
          {error?.message || "An unexpected error occurred."}
        </p>
        <div className="flex justify-center gap-2">
          <Button
            onClick={() => {
              void router.invalidate();
            }}
          >
            <RotateCwIcon className="h-4 w-4 mr-1.5" />
            Try again
          </Button>
          <Button
            variant="outline"
            render={<Link to="/">Go to Today</Link>}
          >
            <HomeIcon className="h-4 w-4 mr-1.5" />
            Go to Today
          </Button>
        </div>
      </div>
    </div>
  );
}

/**
 * Slim variant for the /_auth layout (login, register, password reset).
 * No app shell, no home link — recovery is retry or the auth pages.
 */
export function AuthRouteError({ error }: RouteErrorProps) {
  const router = useRouter();

  if (import.meta.env.DEV) {
    console.error("[auth-route-error] loader/query failed:", error);
  }

  return (
    <div
      role="alert"
      className="flex min-h-svh w-full items-center justify-center p-6"
    >
      <div className="w-full max-w-sm space-y-4 rounded-lg border bg-card p-6 text-center">
        <AlertTriangleIcon className="mx-auto h-8 w-8 text-destructive" />
        <h2 className="text-base font-semibold">Something went wrong</h2>
        <p className="text-sm text-muted-foreground">
          {error?.message || "An unexpected error occurred."}
        </p>
        <Button
          className="w-full"
          onClick={() => {
            void router.invalidate();
          }}
        >
          Try again
        </Button>
      </div>
    </div>
  );
}
