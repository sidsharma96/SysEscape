import type { Room } from "@/lib/graphql/queries";
import { RoomCard } from "./RoomCard";

export function RoomGrid({ rooms }: { rooms: Room[] }) {
  if (rooms.length === 0) {
    return (
      <p className="py-12 text-center text-gray-400">No rooms found.</p>
    );
  }

  return (
    <div className="grid gap-6 grid-cols-1 md:grid-cols-2 lg:grid-cols-3">
      {rooms.map((room) => (
        <RoomCard key={room.slug} room={room} />
      ))}
    </div>
  );
}
