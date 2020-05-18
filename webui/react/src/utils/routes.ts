export const isFullPath = (url: string): boolean => url.startsWith('http');

export const isAbsolutePath = (url: string): boolean => url.startsWith('/');

export const parseUrl = (url: string): URL => {
  let cleanUrl = url;
  if (!isFullPath(url)) {
    if (!isAbsolutePath(url)) cleanUrl = '/' + url;
    cleanUrl = window.location.origin + url;
  }
  return new window.URL(cleanUrl);
};
