import { serverAddress } from 'routes/utils';
import * as Api from 'services/api-ts-sdk';
import { isObject } from 'shared/utils/data';
import handleError from 'utils/error';

import { DetError } from '../shared/utils/error';
import { getResponseStatus, processApiError } from '../shared/utils/service';

/* Response Helpers */

/*
 * This is a failure received from a failed login attempt due to bad credentials
 * 403 is returned by the old API
 * 401 is returned by the new API. We can rely on isAuthFailure
 * when we completely migrate over to the new API.
 */
/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
export const isLoginFailure = (e: any): boolean => {
  const status = getResponseStatus(e);
  return status === 401 || status === 403;
};

/* HTTP Helpers */

/* gRPC Helpers */

export const readStream = async <T = unknown>(
  fetchArgs: Api.FetchArgs,
  onEvent: (event: T) => void,
): Promise<unknown> => {
  try {
    const options = isObject(fetchArgs.options) ? fetchArgs.options : {};

    /*
     * Default fetch credentials is set to `same-origin`, but we need to change it
     * to `include` for local dev because the ports do not match up (3000 vs 8080).
     */
    if (process.env.IS_DEV) options.credentials = 'include';

    const response = await fetch(serverAddress(fetchArgs.url), options);
    if (!response.body) throw new DetError(`Unable to fetch stream from ${fetchArgs.url}.`);

    const decoder = new TextDecoder();
    const reader = response.body.getReader();
    let buffer = '';
    let isCancelled = false;

    // Cancel reader if an abort signal is received.
    if (options?.signal) {
      const signal: AbortSignal = options.signal;
      const abortHandler = () => {
        signal.removeEventListener('abort', abortHandler);
        isCancelled = true;
      };
      signal.addEventListener('abort', abortHandler);
    }

    const handleStreamError = (e: unknown) => handleError(e, { silent: true });
    const handleStreamLine = (line: string) => {
      if (isCancelled) return;
      try {
        const ndjson = JSON.parse(line);
        onEvent(ndjson.result);
      } catch {
        // JSON parsing error occurred, no-op.
      }
    };
    const handleStreamRead = (result: ReadableStreamDefaultReadResult<ArrayBuffer>): unknown => {
      if (isCancelled) return;
      if (result.done) {
        // Process any data buffer remainder.
        buffer = buffer.trim();
        if (buffer.length !== 0) handleStreamLine(buffer);
        return;
      }

      // Append incoming streaming data to buffer.
      buffer += decoder.decode(result.value, { stream: true });

      // Process only newline delimited data buffer.
      const lines = buffer.split('\n');
      for(let i = 0; i < lines.length - 1; ++i) {
        const line = lines[i].trim();
        if (line.length === 0) continue;
        handleStreamLine(line);
      }

      // Keep the unprocessed buffer data.
      buffer = lines[lines.length - 1];

      // Keep reading.
      return reader.read().then(handleStreamRead).catch(handleStreamError);
    };

    return reader.read().then(handleStreamRead).catch(handleStreamError);
  } catch (e) {
    handleError(await processApiError(fetchArgs.url, e));
  }
};
