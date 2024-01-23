import router from 'router';

export const samlUrl = (basePath: string, queries?: string): string => {
  if (queries) {
    queries = queries.replace(/r=0\.\d+[&]?/, '');
    if (queries.includes('&redirect=')) {
      const redirectStartIndex = queries.indexOf('&redirect=');
      const redirectEndIndex = queries.indexOf('&', redirectStartIndex + 1);
      if (redirectEndIndex === -1) {
        queries = queries.slice(0, redirectStartIndex);
      } else {
        queries = queries.slice(0, redirectStartIndex) + queries.slice(redirectEndIndex);
      }
    }
  }
  if (!queries) return basePath;
  return `${basePath}?relayState=${encodeURIComponent(queries)}`;
};

// Decode relayState into expected query params T.
export const handleRelayState = (queries: URLSearchParams): URLSearchParams => {
  const clone = new URLSearchParams(queries);
  if (!queries.has('relayState')) return clone;

  const relayState = new URLSearchParams(clone.get('relayState') || '');
  for (const key of relayState.keys()) {
    // not using entries here in order to handle arrays properly
    const value = relayState.getAll(key);
    clone.set(key, value[0]);
    value.slice(1).forEach((subValue) => clone.append(key, subValue));
  }
  clone.delete('relayState');
  router.getRouter().navigate(`${window.location.pathname}?${clone}`);

  return clone;
};
