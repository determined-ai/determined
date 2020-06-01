const updateFavicon = (iconPath: string): void => {
  const linkEl: HTMLLinkElement | null = document.querySelector("link[rel*='shortcut icon']");
  if (!linkEl) return;
  linkEl.type = 'image/png';
  linkEl.href = iconPath;
};

export const updateFaviconType = (active: boolean): void => {
  const suffixDev = process.env.IS_DEV ? '-dev' : '';
  const suffixActive = active ? '-active' : '';
  updateFavicon(`/favicons/favicon${suffixDev}${suffixActive}.png`);
};

export const getCookie = (name: string): string | null => {
  const regex = new RegExp(`(?:(?:^|.*;\\s*)${name}\\s*\\=\\s*([^;]*).*$)|^.*$`);
  const value = document.cookie.replace(regex, '$1');
  return value ? value : null;
};
