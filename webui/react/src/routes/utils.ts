import { pathToRegexp } from 'path-to-regexp';
import { MouseEvent, MouseEventHandler } from 'react';

import { globalStorage } from 'globalStorage';
import history from 'routes/history';
import { Command, CommandTask, CommandType } from 'types';
import { clone } from 'utils/data';

import routes from './routes';
import { RouteConfig } from './types';

// serverAddress returns determined cluster (master) address.
export const serverAddress = (path = ''): string => {
  if (!!path && isFullPath(path)) return path;

  // Prioritize dynamically set address.
  const customServer = globalStorage.getServerAddress
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

const commandToEventUrl = (command: Partial<Command>): string | undefined => {
  if (command.kind === CommandType.Notebook) return `/notebooks/${command.id}/events`;
  if (command.kind === CommandType.Tensorboard) return `/tensorboard/${command.id}/events?tail=1`;
  return undefined;
};

export const waitPageUrl = (command: Partial<Command>): string | undefined => {
  const eventUrl = commandToEventUrl(command);
  const proxyUrl = command.serviceAddress;
  if (!eventUrl || !proxyUrl) return;
  const event = encodeURIComponent(eventUrl);
  const jump = encodeURIComponent(proxyUrl);
  return `/wait/index.html?event=${event}&jump=${jump}`;
};

export const windowOpenFeatures = [ 'noopener', 'noreferrer' ];

export const openBlank = (url: string): void => {
  window.open(url, '_blank', windowOpenFeatures.join(','));
};

export const openCommand = (command: Command | CommandTask): void => {
  const url = waitPageUrl(command);
  if (!url) throw new Error('command cannot be opened');
  openBlank(process.env.PUBLIC_URL + url);
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

// Given a react url returns the react route path.
const getReactPath = (url: string): string => {
  return parseUrl(url).pathname.replace(process.env.PUBLIC_URL, '');
};

const findReactRoute = (url: string): RouteConfig | undefined => {
  if (isFullPath(url)) {
    if (!url.startsWith(reactHostAddress())) return undefined;
    // Fit it into a relative path
    url = url.replace(reactHostAddress(), '');
  }
  // Check to see if the path matches any of the defined app routes.
  const pathname = getReactPath(url);
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
    history.push(getReactPath(path), { loginRedirect: clone(window.location) });
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
