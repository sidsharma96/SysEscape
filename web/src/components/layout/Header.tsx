import { Link } from "react-router-dom";
import { useViewer } from "@/lib/auth/useViewer";

export function Header() {
  const { viewer, loading, isAuthenticated } = useViewer();

  function handleLogout() {
    fetch("/auth/logout", { method: "POST", credentials: "include" }).then(
      () => {
        window.location.reload();
      },
    );
  }

  return (
    <nav className="border-b border-panel-border bg-surface-mid">
      <div className="mx-auto max-w-7xl px-4 sm:px-6 lg:px-8">
        <div className="flex h-14 items-center justify-between">
          <Link
            to="/"
            className="font-mono text-lg font-semibold tracking-tight"
          >
            Systems Escape Rooms
          </Link>

          <div className="flex items-center gap-6">
            <Link to="/" className="text-sm text-gray-300 hover:text-white">
              Catalog
            </Link>
            <Link
              to="/runs"
              className="text-sm text-gray-300 hover:text-white"
            >
              Runs
            </Link>
          </div>

          <div className="flex items-center gap-4">
            {loading ? (
              <div />
            ) : isAuthenticated ? (
              <>
                <span className="text-sm text-gray-300">
                  {viewer?.githubUsername ?? "User"}
                </span>
                <button
                  type="button"
                  onClick={handleLogout}
                  className="text-sm text-gray-400 hover:text-white"
                >
                  Log out
                </button>
              </>
            ) : (
              <a
                href="/auth/github/login"
                className="text-sm text-gray-300 hover:text-white"
              >
                Sign in with GitHub
              </a>
            )}
          </div>
        </div>
      </div>
    </nav>
  );
}
