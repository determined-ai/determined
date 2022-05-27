import prettyBytes from 'pretty-bytes';

const LETTERS = 'abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ';
const CHARACTERS = `0123456789${LETTERS}`;

export const DEFAULT_ALPHA_NUMERIC_LENGTH = 8;

export const snakeCaseToTitleCase = (text: string): string => {
  const words = text.split('_');
  const capitalizedWords = words.map(word => capitalize(word));
  return capitalizedWords.join(' ');
};

export const camelCaseToKebab = (text: string): string => {
  return text.trim().split('').map((char, index) => {
    return char === char.toUpperCase() ? `${index !== 0 ? '-' : ''}${char.toLowerCase()}` : char;
  }).join('');
};

export const camelCaseToSentence = (text: string): string => {
  const result = text.trim().replace(/([A-Z])/g, ' $1');
  return result.charAt(0).toUpperCase() + result.slice(1);
};

export const kebabToCamelCase = (text: string): string => {
  return text.trim().split('-').map((word, index) => {
    return index === 0 ? word.toLowerCase() : capitalizeWord(word);
  }).join('');
};

export const sentenceToCamelCase = (text: string): string => {
  const result = text.trim().split(' ').map((word, idx) => (
    idx === 0 ? word.toLowerCase() : capitalizeWord(word)
  ));
  return result.join('');
};

export const capitalize = (str: string): string => {
  return str.split(/\s+/).map(part => capitalizeWord(part)).join(' ');
};

export const capitalizeWord = (str: string): string => {
  return str.charAt(0).toUpperCase() + str.slice(1).toLowerCase();
};

export const floatToPercent = (num: number, precision = 2): string => {
  if (isNaN(num)) return 'NaN';
  if (num === Infinity) return 'Infinity';
  if (num === -Infinity) return '-Infinity';
  return (num * 100).toFixed(precision) + '%';
};

export const generateAlphaNumeric = (
  length = DEFAULT_ALPHA_NUMERIC_LENGTH,
  chars = CHARACTERS,
): string => {
  let result = '';
  for (let i = length; i > 0; --i) {
    result += chars[ Math.floor(Math.random() * chars.length) ];
  }
  return result;
};

export const generateUUID = (): string => {
  return [
    generateAlphaNumeric(8),
    generateAlphaNumeric(4),
    generateAlphaNumeric(4),
    generateAlphaNumeric(4),
    generateAlphaNumeric(12),
  ].join('-');
};

export const generateLetters = (length = DEFAULT_ALPHA_NUMERIC_LENGTH): string => {
  return generateAlphaNumeric(length, LETTERS);
};

export const humanReadableBytes = (bytes: number): string => {
  return prettyBytes(bytes);
};

export const listToStr = (list: (string | undefined)[], glue = ' '): string => {
  return list.filter(item => !!item).join(glue);
};

export const toHtmlId = (str: string): string => {
  return str
    .replace(/[\s_]/gi, '-')
    .replace(/[^a-z0-9-]/gi, '')
    .toLowerCase();
};

export const truncate = (str: string, maxLength = 20, suffix = '...'): string => {
  if (maxLength < suffix.length + 1) {
    maxLength = suffix.length + 1;
  }
  if (str.length <= maxLength) {
    return str;
  }
  return str.slice(0, maxLength - suffix.length) + suffix;
};
