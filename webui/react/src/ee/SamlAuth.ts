import queryString from 'query-string';

import history from 'routes/history';

export const samlUrl = (basePath: string, queries?: string): string => {
  if (!queries) return basePath;
  return `${basePath}?relayState=${encodeURIComponent(queries)}`;
};

type WithRelayState<T> = T & {relayState?: string}

// decode relayState into expected query params T
export const handleRelayState = <T>(queries: WithRelayState<T>): T => {
  if (!queries.relayState) return queries;
  queries = {
    ...queries,
    ...(queryString.parse(queries.relayState)),
  };
  delete queries.relayState;
  history.push(`${history.location.pathname}?${queryString.stringify(queries)}`);
  return queries;
};
