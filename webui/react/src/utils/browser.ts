import { parseUrl } from 'routes/utils';
import { getTrialDetails } from 'services/api';
import { V1TrialLogsResponse } from 'services/api-ts-sdk';
import { detApi } from 'services/apiConfig';
import { consumeStream } from 'services/utils';

const updateFavicon = (iconPath: string): void => {
  const linkEl: HTMLLinkElement | null = document.querySelector("link[rel*='shortcut icon']");
  if (!linkEl) return;
  linkEl.type = 'image/png';
  linkEl.href = iconPath;
};

export const updateFaviconType = (active: boolean): void => {
  const suffixDev = process.env.IS_DEV ? '-dev' : '';
  const suffixActive = active ? '-active' : '';
  updateFavicon(`${process.env.PUBLIC_URL}/favicons/favicon${suffixDev}${suffixActive}.png`);
};

export const getCookie = (name: string): string | null => {
  const regex = new RegExp(`(?:(?:^|.*;\\s*)${name}\\s*\\=\\s*([^;]*).*$)|^.*$`);
  const value = document.cookie.replace(regex, '$1');
  return value ? value : null;
};

const downloadBlob = (filename: string, data: Blob): void => {
  const url = window.URL.createObjectURL(data);
  const element = document.createElement('a');
  element.setAttribute('download', filename);
  element.style.display = 'none';
  element.href = url;
  document.body.appendChild(element);
  element.click();
  window.URL.revokeObjectURL(url);
  document.body.removeChild(element);
};

// Different JS engines impose different maximum string lenghts thus for downloading
// large text we split it into different parts and then stich them together as a Blob
export const downloadText = (filename: string, parts: BlobPart[]): void => {
  const data = new Blob(parts, { type: 'text/plain' });
  downloadBlob(filename, data);
};

export const downloadTrialLogs = async (trialId: number): Promise<void> => {
  // String concatnation is fater than array.join https://jsben.ch/OJ3vo
  // characters are utf16 encoded. v8 has different internal representations.
  const MAX_PART_SIZE = 128 * Math.pow(2, 20); // 128m * CHAR_SIZE
  const parts: BlobPart[] = [];
  let downloadStringBuffer = '';
  await consumeStream<V1TrialLogsResponse>(
    detApi.StreamingExperiments.determinedTrialLogs(trialId),
    (ev) => {
      downloadStringBuffer += ev.message;
      if (downloadStringBuffer.length > MAX_PART_SIZE) {
        parts.push(downloadStringBuffer);
        downloadStringBuffer = '';
      }
    },
  );
  if (downloadStringBuffer !== '') parts.push(downloadStringBuffer);
  const trial = await getTrialDetails({ id: trialId });
  downloadText(`experiment_${trial.experimentId}_trial_${trialId}_logs.txt`, parts);
};

const generateLogStringBuffer = (count: number, avgLength: number): string => {
  const msg = new Array(avgLength).fill('a').join('') + '\n';
  let stringBuffer = '';
  for (let i=0; i<count; i++) {
    stringBuffer += (i + msg);
  }
  return stringBuffer;
};

export const simulateLogsDownload = (numCharacters: number): number => {
  const start = Date.now();
  const MAX_PART_SIZE = 128 * Math.pow(2, 20); // 128m * CHAR_SIZE
  const chunkCount = Math.ceil(numCharacters / MAX_PART_SIZE);
  const parts = new Array(chunkCount).fill(0)
    .map(() => generateLogStringBuffer(Math.pow(2, 20), 128));
  downloadText('simulated-logs.txt', parts);
  return (Date.now() - start);
};

/*
 * The method of cache busting here is to send a query string as most
 * modern browsers treat different URLs as different files, causing a
 * request of a fresh copy. The previous method of using `location.reload`
 * with a `forceReload` boolean has been deprecated and not reliable.
 */
export const refreshPage = (): void => {
  const now = Date.now();
  const url = parseUrl(window.location.href);
  url.search = url.search ? `${url.search}&ts=${now}` : `ts=${now}`;
  window.location.href = url.toString();
};
