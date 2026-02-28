import { useQuery } from "urql";
import { ROOMS_QUERY } from "@/lib/graphql/queries";
import type { RoomsQueryResult } from "@/lib/graphql/queries";
import { RoomGrid } from "@/components/catalog/RoomGrid";

function SkeletonCard() {
  return (
    <div className="rounded-lg border border-gray-800 bg-surface-mid p-6 animate-skeleton-delayed">
      <div className="mb-2 h-5 w-3/4 rounded bg-surface-light" />
      <div className="mb-3 flex gap-2">
        <div className="h-5 w-16 rounded-full bg-surface-light" />
        <div className="h-5 w-20 rounded-full bg-surface-light" />
      </div>
      <div className="h-3 w-full rounded bg-surface-light" />
      <div className="mt-2 h-3 w-2/3 rounded bg-surface-light" />
    </div>
  );
}

function SkeletonGrid() {
  return (
    <div className="grid gap-6 grid-cols-1 md:grid-cols-2 lg:grid-cols-3">
      {Array.from({ length: 6 }).map((_, i) => (
        <SkeletonCard key={i} />
      ))}
    </div>
  );
}

export function CatalogPage() {
  const [result] = useQuery<RoomsQueryResult>({ query: ROOMS_QUERY });
  const { data, fetching, error } = result;

  return (
    <div>
      <h1 className="mb-6 text-2xl font-semibold">Room Catalog</h1>
      {fetching && <SkeletonGrid />}
      {error && (
        <p className="text-signal-crit">Failed to load rooms: {error.message}</p>
      )}
      {data && <RoomGrid rooms={data.rooms} />}
    </div>
  );
}
