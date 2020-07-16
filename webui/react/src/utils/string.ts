import prettyBytes from 'pretty-bytes';

export const capitalize = (str: string): string => {
  return str.charAt(0).toUpperCase() + str.slice(1).toLowerCase();
};

export const truncate = (str: string, maxLen: number): string => {
  if (maxLen < 4) {
    str.slice(0, maxLen);
  }
  if (str.length <= maxLen) {
    return str;
  }
  return str.slice(0, maxLen-3) + '...';
};

export const toHtmlId = (str: string): string => {
  return str
    .replace(/[\s_]/gi, '-')
    .replace(/[^a-z0-9-]/gi, '')
    .toLowerCase();
};

export const listToStr = (list: (string|undefined)[], glue = ' '): string => {
  return list.filter(item => !!item).join(glue);
};

export const floatToPercent = (num: number, precision = 2): string => {
  return (num * 100).toFixed(precision) + '%';
};

export const humanReadableFloat = (num: number, precision = 2): string => {
  return num.toExponential(precision);
};

export const humanReadableBytes = (bytes: number): string => {
  return prettyBytes(bytes);
};
