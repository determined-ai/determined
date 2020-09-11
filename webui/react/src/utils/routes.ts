import { MouseEvent, MouseEventHandler } from 'react';

import { routeAll } from 'routes';
import { Command } from 'types';

import { waitPageUrl } from './types';

export const serverAddress = (avoidDevProxy = false): string => {
  if (avoidDevProxy && process.env.IS_DEV) return 'http://localhost:8080';
  return `${window.location.protocol}//${window.location.host}`;
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
