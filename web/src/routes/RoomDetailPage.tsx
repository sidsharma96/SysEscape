import { useParams } from "react-router-dom";

export function RoomDetailPage() {
  const { slug } = useParams<{ slug: string }>();
  return (
    <div>
      <h1 className="mb-4 text-2xl font-semibold">Room: {slug}</h1>
      <p className="text-gray-400">Room detail page placeholder.</p>
    </div>
  );
}
