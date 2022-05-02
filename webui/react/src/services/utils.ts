import { serverAddress } from 'routes/utils';
import * as Api from 'services/api-ts-sdk';
import { isObject } from 'utils/data';
import handleError, {
  DetError, DetErrorOptions, ErrorLevel, ErrorType, isDetError,
} from 'utils/error';

import { DetApi, FetchOptions } from './types';

/* Response Helpers */

/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
const getResponseStatus = (e: any): number | undefined => {
  const errorResponse = e || {};
  return (errorResponse.response || {}).status || errorResponse.status;
};

/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
export const isAuthFailure = (e: any, supportExternalAuth = false): boolean => {
  const status = getResponseStatus(e) ?? 0;
  const authFailureStatuses = [ 401 ];
  if (supportExternalAuth) authFailureStatuses.push(500);
  return authFailureStatuses.includes(status);
};

const isApiResponse = (o: unknown): o is Response => {
  return o instanceof Response;
};

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

/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
export const isNotFound = (e: any): boolean => {
  return getResponseStatus(e) === 404;
};

/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
export const isAborted = (e: any): boolean => {
  return e?.name === 'AbortError';
};

/* HTTP Helpers */

/* Fits API errors into a DetError. */
export const processApiError = async (name: string, e: unknown): Promise<DetError> => {
  const isAuthError = isAuthFailure(e);
  const isApiBadResponse = isDetError(e) && e?.type === ErrorType.ApiBadResponse;
  const options: DetErrorOptions = {
    level: ErrorLevel.Error,
    publicSubject: `Request ${name} failed.`,
    silent: !process.env.IS_DEV || isAuthError || isAborted(e),
    type: ErrorType.Server,
  };

  if (isAuthError) {
    options.publicSubject = `Unauthenticated ${name} request.`;
    options.level = ErrorLevel.Fatal;
    options.type = ErrorType.Auth;
  } else if (isApiBadResponse) {
    options.publicSubject = `Failed in decoding ${name} API response.`;
  }

  if (isApiResponse(e)) {
    try {
      const response = await e.json();
      options.publicMessage = response.error?.error || response.message;
    } catch (err) {
      options.payload = err;
    }
  }

  return new DetError(e, options);
};

export function generateDetApi<Input, DetOutput, Output>(api: DetApi<Input, DetOutput, Output>) {
  return async function(params: Input, options?: FetchOptions): Promise<Output> {
    try {
      const response = api.stubbedResponse ?
        api.stubbedResponse : await api.request(params, options);
      return api.postProcess(response);
    } catch (e) {
      throw (await processApiError(api.name, e));
    }
  };
}

/* gRPC Helpers */

export const readStream = async <T = unknown>(
  fetchArgs: Api.FetchArgs,
  onEvent: (event: T) => void,
): Promise<void> => {
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

    reader.read().then(handleStreamRead).catch(handleStreamError);
  } catch (e) {
    handleError(await processApiError(fetchArgs.url, e));
  }
};

/*
 * This function is primarily used to convert an enum option into a string value
 * that the generated API can take as a request param.
 * More specifically the function takes a value and checks it against a Typescript enum,
 * to make sure the the value is one of the enum option and returns the value as a string.
 */
/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
export const validateDetApiEnum = (enumObject: unknown, value?: unknown): any => {
  if (isObject(enumObject) && value !== undefined) {
    const enumRecord = enumObject as Record<string, string>;
    const stringValue = value as string;
    const validOptions = Object
      .values(enumRecord)
      .filter((_, index) => index % 2 === 0);
    if (validOptions.includes(stringValue)) return stringValue;
    return enumRecord.UNSPECIFIED;
  }
  return undefined;
};

/*
 * This is the same as validateDetApiEnum but validates a list of values.
 * If the validated list is empty, this will return undefined because our
 * API will skip filtering if it sees an `undefined` value for a filter
 * query parameter.
 */
/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
export const validateDetApiEnumList = (enumObject: unknown, values?: unknown[]): any => {
  if (!Array.isArray(values)) return undefined;

  const enumValues = values
    .map(value => validateDetApiEnum(enumObject, value))
    .filter(enumValue => enumValue !== (enumObject as { UNSPECIFIED: unknown }).UNSPECIFIED);
  return enumValues.length !== 0 ? enumValues : undefined;
};

/* eslint-disable-next-line */
export const noOp = (): void => {}
export const identity = <T>(a: T): T => a;
