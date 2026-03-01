import { Navigate } from "react-router-dom";
import { useViewer } from "@/lib/auth/useViewer";

export function LoginPage() {
  const { loading, isAuthenticated } = useViewer();

  if (loading) {
    return null;
  }

  if (isAuthenticated) {
    return <Navigate to="/" />;
  }

  return (
    <div className="flex min-h-[60vh] items-center justify-center">
      <div className="rounded-lg border border-panel-border bg-surface-mid p-8 text-center">
        <h1 className="mb-4 text-2xl font-semibold">Welcome</h1>
        <p className="mb-6 text-gray-400">
          Sign in to access Systems Escape Rooms.
        </p>
        <a
          href="/auth/github/login"
          className="inline-block rounded bg-gray-800 px-6 py-3 text-sm font-medium text-white hover:bg-gray-700"
        >
          Sign in with GitHub
        </a>
      </div>
    </div>
  );
}
