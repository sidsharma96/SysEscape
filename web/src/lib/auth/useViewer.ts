import { useQuery } from "urql";
import type { ViewerQueryResult } from "@/lib/graphql/queries";
import { VIEWER_QUERY } from "@/lib/graphql/queries";

export function useViewer() {
  const [result] = useQuery<ViewerQueryResult>({ query: VIEWER_QUERY });

  return {
    viewer: result.data?.viewer ?? null,
    loading: result.fetching,
    isAuthenticated: !result.fetching && result.data?.viewer != null,
  };
}
