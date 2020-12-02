import { pathToRegexp } from 'path-to-regexp';
import { MouseEvent, MouseEventHandler } from 'react';

import { globalStorage } from 'globalStorage';
import history from 'routes/history';
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

export const handlePath = (
  event: MouseEvent,
  options: {
    onClick?: MouseEventHandler,
    path?: string,
    popout?: boolean,
    external?: boolean,
  } = {},
): void => {
  event.persist();
  event.preventDefault();

  const href = options.path ? linkPath(options.path, options.external) : undefined;

  if (options.onClick) {
    options.onClick(event);
  } else if (href) {
    if (event.button === 1 || event.metaKey || event.ctrlKey || options.popout) {
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

const findReactRoute = (url: string): RouteConfig | undefined => {
  if (isFullPath(url)) {
    if (!url.startsWith(reactHostAddress())) return undefined;
    // Fit it into a relative path
    url = url.replace(reactHostAddress(), '');
  }
  // Check to see if the path matches any of the defined app routes.
  const pathname = parseUrl(url).pathname.replace(process.env.PUBLIC_URL, '');
  return routes
    .filter(route => route.path !== '*')
    .find(route => {
      const routeRegex = pathToRegexp(route.path);
      return routeRegex.test(pathname);
    });
};

const routeToExternalUrl = (path: string): void => {
  window.location.assign(path);
};

export const routeAll = (path: string): void => {
  const matchingReactRoute = findReactRoute(path);
  if (!matchingReactRoute) {
    routeToExternalUrl(path);
  } else {
    history.push(stripUrl(path), { loginRedirect: clone(window.location) });
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
