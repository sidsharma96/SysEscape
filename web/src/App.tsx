import { lazy, Suspense } from "react";
import { BrowserRouter, Routes, Route } from "react-router-dom";
import { Provider } from "urql";
import { urqlClient } from "@/lib/graphql/client";
import { Layout } from "./routes/Layout.tsx";
import { CatalogPage } from "./routes/CatalogPage.tsx";
import { RoomDetailPage } from "./routes/RoomDetailPage.tsx";
import { LoginCallbackPage } from "./routes/LoginCallbackPage.tsx";
import { RunsPage } from "./routes/RunsPage.tsx";
import { AdminPublishPage } from "./routes/AdminPublishPage.tsx";

const EngineAPage = lazy(() =>
  import("./routes/EngineAPage.tsx").then((m) => ({ default: m.EngineAPage })),
);
const EngineBPage = lazy(() =>
  import("./routes/EngineBPage.tsx").then((m) => ({ default: m.EngineBPage })),
);

function LoadingFallback() {
  return (
    <div className="flex items-center justify-center min-h-[50vh]">
      <p className="text-gray-400">Loading...</p>
    </div>
  );
}

export function App() {
  return (
    <BrowserRouter>
      <Provider value={urqlClient}>
        <Routes>
          <Route element={<Layout />}>
            <Route index element={<CatalogPage />} />
            <Route path="rooms/:slug" element={<RoomDetailPage />} />
            <Route path="login/callback" element={<LoginCallbackPage />} />
            <Route
              path="play/:runId/engine-a"
              element={
                <Suspense fallback={<LoadingFallback />}>
                  <EngineAPage />
                </Suspense>
              }
            />
            <Route
              path="play/:runId/engine-b"
              element={
                <Suspense fallback={<LoadingFallback />}>
                  <EngineBPage />
                </Suspense>
              }
            />
            <Route path="runs" element={<RunsPage />} />
            <Route path="admin/publish" element={<AdminPublishPage />} />
          </Route>
        </Routes>
      </Provider>
    </BrowserRouter>
  );
}
