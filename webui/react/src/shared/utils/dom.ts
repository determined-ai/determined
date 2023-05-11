import ansiConverter from 'ansi-to-html';

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
    // This method is only available on https and localhost
    await navigator.clipboard.writeText(content);
  } catch (e) {
    throw new Error('Clipboard access on https and localhost only!');
  }
};

export const readFromClipboard = async (): Promise<string> => {
  try {
    // This method is only available on https and localhost
    return await navigator.clipboard.readText();
  } catch (e) {
    throw new Error('Clipboard access on https and localhost only!');
  }
};
