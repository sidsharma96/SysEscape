import { Link } from "react-router-dom";
import type { Room } from "@/lib/graphql/queries";
import { DifficultyBadge } from "./DifficultyBadge";

export function RoomCard({ room }: { room: Room }) {
  return (
    <Link
      to={`/rooms/${room.slug}`}
      className="block rounded-lg border border-gray-800 bg-panel-bg p-6 transition-colors hover:border-gray-600"
    >
      <h3 className="mb-2 text-lg font-semibold text-gray-100">{room.title}</h3>
      <div className="mb-3 flex flex-wrap items-center gap-2">
        <span className="rounded-full bg-surface-light px-2 py-0.5 text-xs font-medium text-gray-300">
          {room.district}
        </span>
        <span className="rounded-full bg-surface-light px-2 py-0.5 text-xs font-medium text-gray-300">
          Engine {room.engine}
        </span>
        <DifficultyBadge difficulty={room.difficulty} />
      </div>
      <p className="line-clamp-2 text-sm text-gray-400">{room.description}</p>
    </Link>
  );
}
