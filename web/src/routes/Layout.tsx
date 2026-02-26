import { Link, Outlet } from "react-router-dom";

export function Layout() {
  return (
    <div className="min-h-screen bg-surface-dark text-gray-100">
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
          </div>
        </div>
      </nav>
      <main className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
        <Outlet />
      </main>
    </div>
  );
}
