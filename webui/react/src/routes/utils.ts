import { pathToRegexp } from 'path-to-regexp';
import { MouseEvent, MouseEventHandler } from 'react';

import history from 'routes/history';
import { Command } from 'types';
import { clone } from 'utils/data';

import { waitPageUrl } from '../utils/types';

import { appRoutes } from './routes';

export const serverAddress = (path: string): string => {
  const { host, protocol } = window.location;
  const address = process.env.IS_DEV ? 'http://localhost:8080' : `${protocol}//${host}`;
  return address + path;
};

export const isFullPath = (url: string): boolean => url.startsWith('http');

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

export const openCommand = (command: Command): void => {
  const url = waitPageUrl(command);
  if (!url) throw new Error('command cannot be opened');
  openBlank(url);
};

export const handlePath = (
  event: MouseEvent,
  options: {
    onClick?: MouseEventHandler,
    path?: string,
    popout?: boolean,
  } = {},
): void => {
  event.persist();
  event.preventDefault();

  if (options.onClick) {
    options.onClick(event);
  } else if (options.path) {
    if (event.button === 1 || event.metaKey || event.ctrlKey || options.popout) {
      openBlank(options.path);
    } else {
      routeAll(options.path);
    }
  }
};

// Is the path going to be served from the same host?
const isDetRoute = (url: string): boolean => {
  if (!isFullPath(url)) return true;
  if (process.env.IS_DEV) {
    // dev live is served on a different port
    return parseUrl(url).hostname === window.location.hostname;
  }
  return parseUrl(url).host === window.location.host;
};

const isReactRoute = (url: string): boolean => {
  if (!isDetRoute(url)) return false;

  // Check to see if the path matches any of the defined app routes.
  const pathname = parseUrl(url).pathname;
  return !!appRoutes
    .filter(route => route.path !== '*')
    .find(route => {
      return route.exact ? pathname === route.path : !!pathToRegexp(route.path).exec(pathname);
    });
};

const routeToExternalUrl = (path: string): void => {
  window.location.assign(path);
};

export const routeAll = (path: string): void => {
  if (!isReactRoute(path)) {
    routeToExternalUrl(path);
  } else {
    history.push(path, { loginRedirect: clone(window.location) });
  }
};
