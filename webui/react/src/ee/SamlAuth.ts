import queryString from 'query-string';

import router from 'router';
import { clone } from 'shared/utils/data';

export const samlUrl = (basePath: string, queries?: string): string => {
  if (!queries) return basePath;
  return `${basePath}?relayState=${encodeURIComponent(queries)}`;
};

type WithRelayState<T> = T & { relayState?: string };

// Decode relayState into expected query params T.
export const handleRelayState = <T>(queries: WithRelayState<T>): T => {
  if (!queries.relayState) return clone(queries);

  const newQueries = {
    ...queries,
    ...queryString.parse(queries.relayState),
  };
  delete newQueries.relayState;
  router.getRouter().navigate(`${window.location.pathname}?${queryString.stringify(newQueries)}`);

  return newQueries;
};
