export function CatalogPage() {
  return (
    <div>
      <h1 className="mb-6 text-2xl font-semibold">Room Catalog</h1>
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
        {[1, 2, 3].map((i) => (
          <div
            key={i}
            className="rounded-lg border border-panel-border bg-panel-bg p-6 transition-colors hover:bg-panel-hover"
          >
            <div className="mb-2 h-4 w-3/4 rounded bg-surface-light" />
            <div className="h-3 w-1/2 rounded bg-surface-light" />
          </div>
        ))}
      </div>
    </div>
  );
}
