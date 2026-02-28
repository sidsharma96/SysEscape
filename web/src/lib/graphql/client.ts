import { createClient, cacheExchange, fetchExchange } from "urql";

export const urqlClient = createClient({
  url: import.meta.env.VITE_GRAPHQL_URL || "/graphql",
  preferGetMethod: false,
  fetchOptions: { credentials: "include" },
  exchanges: [cacheExchange, fetchExchange],
});
