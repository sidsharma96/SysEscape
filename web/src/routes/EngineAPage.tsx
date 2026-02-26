import { useParams } from "react-router-dom";

export function EngineAPage() {
  const { runId } = useParams<{ runId: string }>();
  return (
    <div>
      <h1 className="mb-4 text-2xl font-semibold">Engine A</h1>
      <p className="text-gray-400">Run: {runId}</p>
      <p className="text-gray-400">Engine A gameplay page placeholder.</p>
    </div>
  );
}
