export const isFullPath = (url: string): boolean => {
  return url.startsWith('http');
};

export const parseUrl = (url: string): URL => {
  if (!isFullPath(url)) {
    if (!url.startsWith('/')) url = '/' + url; // TODO assume url is absolute, or we could throw
    url = window.location.origin + url;
  }
  return new window.URL(url);
};
