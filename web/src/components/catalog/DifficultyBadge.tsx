import type { Room } from "@/lib/graphql/queries";

const difficultyStyles: Record<Room["difficulty"], string> = {
  L0: "bg-signal-ok/20 text-signal-ok",
  L1: "bg-signal-warn/20 text-signal-warn",
  L2: "bg-amber-500/20 text-amber-500",
  L3: "bg-signal-crit/20 text-signal-crit",
};

const difficultyLabels: Record<Room["difficulty"], string> = {
  L0: "L0 – Guided",
  L1: "L1 – Standard",
  L2: "L2 – Hard",
  L3: "L3 – Expert",
};

export function DifficultyBadge({ difficulty }: { difficulty: Room["difficulty"] }) {
  return (
    <span
      className={`rounded-full px-2 py-0.5 text-xs font-medium ${difficultyStyles[difficulty]}`}
    >
      {difficultyLabels[difficulty]}
    </span>
  );
}
