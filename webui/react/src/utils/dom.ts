/* eslint-disable @typescript-eslint/no-var-requires */
const ansiConverter = require('ansi-to-html');
const converter = new ansiConverter({ newline: true });

export const ansiToHtml = (ansi: string): string => {
  const ansiWithHtml = ansi
    .replace(/(&|\u0026)/g, '&amp;')
    .replace(/(>|\u003e)/g, '&gt;')
    .replace(/(<|\u003c)/g, '&lt;')
    .replace(/('|\u0027)/g, '&apos;')
    .replace(/("|\u0022)/g, '&quot;');
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
