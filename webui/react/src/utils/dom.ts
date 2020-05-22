export const toRem = (px?: number | string): string => {
  if (px == null) return 'auto';
  if (typeof px === 'number') return `${px / 10}rem`;

  const matches = px.match(/(\d+\.?\d*)\s*(px|rem)/i);
  if (matches?.length === 3) {
    const type = matches[2];
    const value = parseFloat(matches[1]);
    if (type === 'px') return `${value / 10}rem`;
    if (type === 'rem') return `${value}rem`;
  }

  return px;
};

/* eslint-disable @typescript-eslint/no-var-requires */
const ansiConverter = require('ansi-to-html');
const converter = new ansiConverter({ newline: true });

export const ansiToHtml = (ansi: string): string => {
  return converter.toHtml(ansi);
};
