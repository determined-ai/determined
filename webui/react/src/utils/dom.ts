import { notification } from 'antd';

/*
 * Calculates REM based on base font size of 62.5%.
 * This causes the ratio between REM to px to be 1 to 10.
 * So 1rem becomes 10px.
 */
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
  const ansiWithHtml = ansi.replace('<', '&lt;').replace('>', '&gt;');
  return converter.toHtml(ansiWithHtml);
};

export const copyToClipboard = async (content: string): Promise<void> => {
  if (navigator.clipboard) {
    try {
      return await navigator.clipboard.writeText(content);
    } catch (e) {
      return Promise.reject(Error('Unable to write to clipboard.'));
    }
  }
  return Promise.reject(Error('Writing to clipboard is only available on https and localhost.'));
};

export const readFromClipboard = async (): Promise<string> => {
  if (navigator.clipboard) {
    try {
      return await navigator.clipboard.readText();
    } catch (e) {
      return Promise.reject(Error('Unable to read from clipboard.'));
    }
  }
  return Promise.reject(Error('Reading from clipboard is only available on https and localhost.'));
};
