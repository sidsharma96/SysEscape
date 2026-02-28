import { useParams } from "react-router-dom";
import { useQuery } from "urql";
import { ROOM_BY_SLUG_QUERY } from "@/lib/graphql/queries";
import type { RoomBySlugQueryResult } from "@/lib/graphql/queries";
import { DifficultyBadge } from "@/components/catalog/DifficultyBadge";

function DetailSkeleton() {
  return (
    <div className="animate-skeleton-delayed">
      {/* Title */}
      <div className="mb-4 h-8 w-2/5 rounded bg-surface-light" />
      {/* Badges row */}
      <div className="mb-6 flex gap-2">
        <div className="h-6 w-28 rounded-full bg-surface-light" />
        <div className="h-6 w-24 rounded-full bg-surface-light" />
        <div className="h-6 w-28 rounded-full bg-surface-light" />
      </div>
      {/* Description block */}
      <div className="mb-8 space-y-2.5">
        <div className="h-4 w-full rounded bg-surface-light" />
        <div className="h-4 w-11/12 rounded bg-surface-light" />
        <div className="h-4 w-5/6 rounded bg-surface-light" />
        <div className="h-4 w-3/4 rounded bg-surface-light" />
      </div>
      {/* Button */}
      <div className="h-10 w-32 rounded-lg bg-surface-light" />
    </div>
  );
}

export function RoomDetailPage() {
  const { slug } = useParams<{ slug: string }>();
  const [result] = useQuery<RoomBySlugQueryResult>({
    query: ROOM_BY_SLUG_QUERY,
    variables: { slug },
    pause: !slug,
  });
  const { data, fetching, error } = result;

  if (error) {
    return <p className="text-signal-crit">Failed to load room: {error.message}</p>;
  }

  if (fetching || !data) return <DetailSkeleton />;

  const room = data.roomBySlug;
  if (!room) {
    return <p className="text-gray-400">Room not found.</p>;
  }

  return (
    <div>
      <h1 className="mb-4 text-2xl font-semibold">{room.title}</h1>
      <div className="mb-6 flex flex-wrap items-center gap-2">
        <span className="rounded-full bg-surface-light px-2 py-0.5 text-xs font-medium text-gray-300">
          {room.district}
        </span>
        <span className="rounded-full bg-surface-light px-2 py-0.5 text-xs font-medium text-gray-300">
          Engine {room.engine}
        </span>
        <DifficultyBadge difficulty={room.difficulty} />
      </div>
      <p className="mb-8 text-gray-300">{room.description}</p>
      <button
        disabled
        className="rounded-lg bg-surface-light px-6 py-2 text-sm font-medium text-gray-500 cursor-not-allowed"
      >
        Start Run
      </button>
    </div>
  );
}
