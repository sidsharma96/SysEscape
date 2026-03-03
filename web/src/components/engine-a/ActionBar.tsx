import { memo } from "react";

interface ActionBarProps {
  actions: string[];
  appliedActions: string[];
  onDispatch: (actionKey: string) => void;
}

export const ActionBar = memo(function ActionBar({ actions, appliedActions, onDispatch }: ActionBarProps) {
  return (
    <div className="flex flex-wrap gap-2">
      {actions.map((actionKey) => {
        const applied = appliedActions.includes(actionKey);
        return (
          <button
            key={actionKey}
            disabled={applied}
            onClick={() => onDispatch(actionKey)}
            className={
              applied
                ? "rounded-lg bg-surface-light px-4 py-2 text-sm font-medium text-gray-500 cursor-not-allowed"
                : "rounded-lg bg-surface-mid px-4 py-2 text-sm font-medium text-gray-100 hover:bg-panel-hover"
            }
          >
            {actionKey}
          </button>
        );
      })}
    </div>
  );
});
