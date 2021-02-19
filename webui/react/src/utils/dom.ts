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

/*
 * Calculates Pixel from Rem based on base font size of 62.5%.
 * This causes the ratio between REM to px to be 1 to 10.
 * So 1rem becomes 10px.
 */
export const toPixel = (rem?: number | string): string => {
  if (rem == null) return 'auto';
  if (typeof rem === 'number') return `${rem * 10}px`;

  const matches = rem.match(/(\d+\.?\d*|\.\d+)\s*(px|rem)/i);
  if (matches?.length === 3) {
    const type = matches[2];
    const value = parseFloat(matches[1]);
    if (type === 'rem') return `${value * 10}px`;
    if (type === 'px') return `${value}px`;
  }

  return rem;
};

/* eslint-disable @typescript-eslint/no-var-requires */
const ansiConverter = require('ansi-to-html');
const converter = new ansiConverter({ newline: true });

export const ansiToHtml = (ansi: string): string => {
  const ansiWithHtml = ansi.replace('<', '&lt;').replace('>', '&gt;');
  return converter.toHtml(ansiWithHtml);
};

export const copyToClipboard = async (content: string): Promise<void> => {
  try {
    if (navigator.clipboard) {
      // This method is only available on https and localhost
      await navigator.clipboard.writeText(content);
    } else if (document.body && document.execCommand) {
      // This is a fallback but deprecated method
      const textarea = document.createElement('textarea');
      textarea.id = 'clipboard';
      document.body.appendChild(textarea);
      textarea.value = content;
      textarea.select();
      document.execCommand('copy');
      textarea.parentNode?.removeChild(textarea);
    } else {
      throw new Error();
    }
    return;
  } catch (e) {
    return Promise.reject(new Error('Unable to write to clipboard.'));
  }
};

export const readFromClipboard = async (): Promise<string> => {
  try {
    let content = '';
    if (navigator.clipboard) {
      // This method is only available on https and localhost
      content = await navigator.clipboard.readText();
    } else if (document.body && document.execCommand) {
      // This is a fallback but deprecated method
      const textarea = document.createElement('textarea');
      textarea.id = 'clipboard';
      document.body.appendChild(textarea);
      textarea.select();
      document.execCommand('paste');
      content = textarea.value;
      textarea.parentNode?.removeChild(textarea);
    } else {
      throw new Error();
    }
    return content;
  } catch (e) {
    return Promise.reject(new Error('Unable to read from clipboard.'));
  }
};
