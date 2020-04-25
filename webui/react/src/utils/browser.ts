export const updateFavicon = (iconPath: string): void => {
  const linkEl: HTMLLinkElement | null = document.querySelector("link[rel*='shortcut icon']");
  if (!linkEl) return;
  linkEl.type = 'image/x-icon';
  linkEl.rel = 'shortcut icon';
  linkEl.href = iconPath;
};
