import { isObject } from 'components/kit/internal/functions';
import { ErrorHandler, FetchArgs } from 'components/kit/internal/types';

export const readLogStream = async <T = unknown>(
  serverAddress: (path: string) => string,
  fetchArgs: FetchArgs,
  onError: ErrorHandler,
  onEvent?: (event: T) => void,
): Promise<unknown> => {
  try {
    const options = isObject(fetchArgs.options) ? fetchArgs.options : {};

    /*
     * Default fetch credentials is set to `same-origin`, but we need to change it
     * to `include` for local dev because the ports do not match up (3000 vs 8080).
     */
    if (process.env.IS_DEV) options.credentials = 'include';

    const response = await fetch(serverAddress(fetchArgs.url), options);

    if (!response.ok) {
      const body = await response.json();
      const e = new Error(body?.error?.message);
      onError(e);
      return;
    }

    if (!response.body) return onError(`Unable to fetch stream from ${fetchArgs.url}.`);

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

    const handleStreamError = (e: Error) => onError(e);
    const handleStreamLine = (line: string) => {
      if (isCancelled) return;
      try {
        const ndjson = JSON.parse(line);
        if (ndjson.error) {
          onError(ndjson.error);
        } else {
          onEvent?.(ndjson.result);
        }
      } catch {
        // JSON parsing error occurred, no-op.
      }
    };
    const handleStreamRead = (result: ReadableStreamReadResult<ArrayBuffer>): unknown => {
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
      for (let i = 0; i < lines.length - 1; ++i) {
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
    onError(e);
  }
};
