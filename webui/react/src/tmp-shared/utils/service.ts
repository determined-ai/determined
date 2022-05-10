/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
import { DetApi, FetchOptions } from '../types';

import { isObject } from './data';
import { DetError, DetErrorOptions, ErrorLevel, ErrorType, isDetError } from './error';

/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
export const getResponseStatus = (e: any): number | undefined => {
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
/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
export const isNotFound = (e: any): boolean => {
  return getResponseStatus(e) === 404;
};
/* eslint-disable-next-line @typescript-eslint/no-explicit-any */
export const isAborted = (e: any): boolean => {
  return e?.name === 'AbortError';
};
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
  return async function (params: Input, options?: FetchOptions): Promise<Output> {
    try {
      const response = api.stubbedResponse ?
        api.stubbedResponse : await api.request(params, options);
      return api.postProcess(response);
    } catch (e) {
      throw (await processApiError(api.name, e));
    }
  };
}

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
export const noOp = (): void => {
};
export const identity = <T>(a: T): T => a;
