import { Outlet } from "react-router-dom";
import { Header } from "@/components/layout/Header";

export function Layout() {
  return (
    <div className="min-h-screen bg-surface-dark text-gray-100">
      <Header />
      <main className="mx-auto max-w-7xl px-4 py-8 sm:px-6 lg:px-8">
        <Outlet />
      </main>
    </div>
  );
}
