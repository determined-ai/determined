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
