import { pathToRegexp } from 'path-to-regexp';
import queryString from 'query-string';
import React from 'react';

import { globalStorage } from 'globalStorage';
import history from 'routes/history';
import { ClusterApi, Configuration } from 'services/api-ts-sdk';
import { CommandTask } from 'types';
import { clone } from 'utils/data';

import routes from './routes';
import { RouteConfig } from './types';

// serverAddress returns determined cluster (master) address.
export const serverAddress = (path = ''): string => {
  if (!!path && isFullPath(path)) return path;

  // Prioritize dynamically set address.
  const customServer = globalStorage.serverAddress
    || process.env.SERVER_ADDRESS as string;

  return (customServer || reactHostAddress()) + path;
};

// checks to see if the provided address resolves to a live Determeind server or not.
export const checkServerAlive = async (address?: string): Promise<boolean> => {
  address = address || serverAddress();
  try {
    const clusterApi = new ClusterApi(new Configuration({ basePath: address }));
    const data = await clusterApi.determinedGetMaster();
    const attrs = [ 'version', 'masterId', 'clusterId' ];
    for (const attr of attrs) {
      // The server doesn't look like a determined server.
      if (!(attr in data)) return false;
    }
    return true;
  } catch (_) {
    return false;
  }
};

// Returns the address to the server hosting react assets
// excluding the path to the subdirectory if any.
export const reactHostAddress = (): string => {
  return `${window.location.protocol}//${window.location.host}`;
};

export const isFullPath = (url: string): boolean => url.startsWith('http');

// whether the input is pathed from / or not.
export const isAbsolutePath = (url: string): boolean => url.startsWith('/');

export const ensureAbsolutePath = (url: string): string => isAbsolutePath(url) ? url : '/' + url;

export const parseUrl = (url: string): URL => {
  let cleanUrl = url;
  if (!isFullPath(url)) {
    cleanUrl = ensureAbsolutePath(url);
    cleanUrl = window.location.origin + url;
  }
  return new window.URL(cleanUrl);
};

export const locationToPath = (location?: Location): string | null => {
  if (!location || !location.pathname) return null;
  return location.pathname + location.search + location.hash;
};

export const windowOpenFeatures = [ 'noopener', 'noreferrer' ];

export const openBlank = (url: string): void => {
  window.open(url, '_blank', windowOpenFeatures.join(','));
};

export type AnyMouseEvent = MouseEvent | React.MouseEvent;
export type AnyMouseEventHandler = (event: AnyMouseEvent) => void;
export const isMouseEvent = (
  ev: AnyMouseEvent | React.KeyboardEvent,
): ev is AnyMouseEvent => {
  return 'button' in ev;
};
export const isNewTabClickEvent = (event: AnyMouseEvent): boolean => {
  return event.button === 1 || event.metaKey || event.ctrlKey;
};

export const handlePath = (
  event: AnyMouseEvent,
  options: {
    external?: boolean,
    onClick?: AnyMouseEventHandler,
    path?: string,
    popout?: boolean,
  } = {},
): void => {
  // FIXME As of v17, e.persist() doesn’t do anything because the SyntheticEvent is no longer
  // pooled.
  // event.persist();
  event.preventDefault();

  const href = options.path ? linkPath(options.path, options.external) : undefined;

  if (options.onClick) {
    options.onClick(event);
  } else if (href) {
    if (isNewTabClickEvent(event) || options.popout) {
      openBlank(href);
    } else {
      routeAll(href);
    }
  }
};

// remove host and public_url.
const stripUrl = (aUrl: string): string => {
  const url = parseUrl(aUrl);
  const rest = url.href.replace(url.origin, '');
  if (rest.startsWith(process.env.PUBLIC_URL)) {
    return rest.replace(process.env.PUBLIC_URL, '');
  }
  return rest;
};

export const findReactRoute = (url: string): RouteConfig | undefined => {
  if (isFullPath(url)) {
    if (!url.startsWith(reactHostAddress())) return undefined;
    // Fit it into a relative path
    url = url.replace(reactHostAddress(), '');
  }
  if (!url.startsWith(process.env.PUBLIC_URL)) {
    return undefined;
  }
  // Check to see if the path matches any of the defined app routes.
  const pathname = url.replace(process.env.PUBLIC_URL, '');
  return routes
    .filter(route => route.path !== '*')
    .find(route => {
      const routeRegex = pathToRegexp(route.path);
      return routeRegex.test(pathname);
    });
};

export const routeToExternalUrl = (path: string): void => {
  window.location.assign(path);
};
export const routeToReactUrl = (path: string): void => {
  history.push(stripUrl(path), { loginRedirect: clone(window.location) });
};

/*
  routeAll determines whether a path should be routed through internal React router or hanled
  by the browser.
  input `path` should include the PUBLIC_URL if there is one set. eg if react is being served
  in a subdirectory.
*/
export const routeAll = (path: string): void => {
  const matchingReactRoute = findReactRoute(path);
  if (!matchingReactRoute) {
    routeToExternalUrl(path);
  } else {
    routeToReactUrl(path);
  }
};

export const linkPath = (aPath: string, external = false): string => {
  if (isFullPath(aPath)) return aPath;
  let path;
  if (external) {
    if (isAbsolutePath(aPath)) {
      path = serverAddress() + aPath;
    } else {
      path = aPath;
    }
  } else {
    path = process.env.PUBLIC_URL + aPath;
  }
  return path;
};

export const paths = {
  cluster: (): string => {
    return '/cluster';
  },
  dashboard: (): string => {
    return '/dashboard';
  },
  docs: (suffix?: string): string => {
    return `/docs${suffix || ''}`;
  },
  experimentDetails: (experimentId: number | string): string => {
    return `/experiments/${experimentId}`;
  },
  experimentList: (): string => {
    return '/experiments';
  },
  experimentModelDef: (experimentId: number | string): string => {
    return `/experiments/${experimentId}/model_def`;
  },
  login: (): string => {
    return '/login';
  },
  logout: (): string => {
    return '/logout';
  },
  masterLogs: (): string => {
    return '/logs';
  },
  reload: (path: string): string => {
    return `/reload?${queryString.stringify({ path })}`;
  },
  taskList: (): string => {
    return '/tasks';
  },
  taskLogs: (task: CommandTask): string => {
    const taskType = task.type.toLocaleLowerCase();
    return`/${taskType}/${task.id}/logs?id=${task.name}`;
  },
  trialDetails: (trialId: number | string, experimentId?: number | string): string => {
    if (!experimentId) {
      return `/trials/${trialId}`;
    }
    return `/experiments/${experimentId}/trials/${trialId}`;
  },
  trialLogs: (trialId: number | string, experimentId: number | string): string => {
    return `/experiments/${experimentId}/trials/${trialId}/logs`;
  },
  users: (): string => {
    return '/users';
  },
};
