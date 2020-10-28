import queryString from 'query-string';

import history from 'routes/history';
import { clone } from 'utils/data';

export const samlUrl = (basePath: string, queries?: string): string => {
  if (!queries) return basePath;
  return `${basePath}?relayState=${encodeURIComponent(queries)}`;
};

type WithRelayState<T> = T & {relayState?: string}

// decode relayState into expected query params T
export const handleRelayState = <T>(queries: WithRelayState<T>): T => {
  if (!queries.relayState) return clone(queries);

  const newQueries = {
    ...queries,
    ...(queryString.parse(queries.relayState)),
  };
  delete newQueries.relayState;
  history.push(`${history.location.pathname}?${queryString.stringify(newQueries)}`);

  return newQueries;
};
