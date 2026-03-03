import { useState } from "react";
import { Link } from "react-router-dom";
import { useViewer } from "@/lib/auth/useViewer";

export function Header() {
  const { viewer, loading, isAuthenticated } = useViewer();
  const [menuOpen, setMenuOpen] = useState(false);

  function handleLogout() {
    fetch("/auth/logout", { method: "POST", credentials: "include" }).then(
      () => {
        window.location.reload();
      },
    );
  }

  const authContent = loading ? (
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
  );

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

          <div className="hidden md:flex items-center gap-6">
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

          <div className="hidden md:flex items-center gap-4">
            {authContent}
          </div>

          <button
            type="button"
            className="md:hidden"
            aria-label="Toggle menu"
            onClick={() => setMenuOpen((prev) => !prev)}
          >
            <svg
              className="h-6 w-6 text-gray-300"
              fill="none"
              viewBox="0 0 24 24"
              stroke="currentColor"
              strokeWidth={2}
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                d="M4 6h16M4 12h16M4 18h16"
              />
            </svg>
          </button>
        </div>

        {menuOpen && (
          <div
            data-testid="mobile-menu"
            className="md:hidden border-t border-panel-border pb-3 space-y-1"
          >
            <Link
              to="/"
              onClick={() => setMenuOpen(false)}
              className="block py-2 px-4 text-sm text-gray-300 hover:text-white hover:bg-surface-light"
            >
              Catalog
            </Link>
            <Link
              to="/runs"
              onClick={() => setMenuOpen(false)}
              className="block py-2 px-4 text-sm text-gray-300 hover:text-white hover:bg-surface-light"
            >
              Runs
            </Link>
            <div className="px-4 py-2 flex items-center gap-4">
              {authContent}
            </div>
          </div>
        )}
      </div>
    </nav>
  );
}
