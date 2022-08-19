import React from 'react';

import history from '../routes/history';

import { clone } from './data';
import rootLogger from './Logger';

const logger = rootLogger.extend('utils', 'routes');

export const isFullPath = (url: string): boolean => {
  try {
    return url.startsWith('http') && !!(new URL(url));
  } catch (e){
    return false;
  }
};
// whether the input is pathed from / or not.
export const isAbsolutePath = (url: string): boolean => {
  const regex = /^\/(\w+\/)*\w*$/i;
  return regex.test(url);
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
// Returns the address to the server hosting react assets
// excluding the path to the subdirectory if any.
export const reactHostAddress = (): string => {
  return `${window.location.protocol}//${window.location.host}`;
};
export const ensureAbsolutePath = (url: string): string => isAbsolutePath(url) ? url : '/' + url;
export const filterOutLoginLocation = (
  location: { pathname: string },
): { pathname: string } | undefined => {
  return location.pathname.includes('login') ? undefined : clone(location);
};
export const parseUrl = (url: string): URL => {
  let cleanUrl = url;
  if (!isFullPath(url)) {
    cleanUrl = ensureAbsolutePath(url);
    cleanUrl = window.location.origin + url;
  }
  return new window.URL(cleanUrl);
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
export const routeToExternalUrl = (path: string): void => {
  logger.trace('routing to external url', path);
  window.location.assign(path);
};
export const routeToReactUrl = (path: string): void => {
  logger.trace('routing to react url', path);
  history.push(stripUrl(path), { loginRedirect: filterOutLoginLocation(window.location) });
};
