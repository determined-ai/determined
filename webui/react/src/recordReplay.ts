/* eslint-disable no-console */

import { sha512 } from 'js-sha512';

import handleError, { ErrorType } from 'ErrorHandler';
import { globalStorage } from 'globalStorage';
import { downloadText } from 'utils/browser';
import { clone } from 'utils/data';
import { Storage } from 'utils/storage';
import { Eventually } from 'utils/types';

const rrStorage = new Storage({
  basePath: 'recordReplay',
  store: window.localStorage,
});

const rrConfigStorage = rrStorage.fork('config');
const rrResponseStorage = rrStorage.fork('responses');

export const userPreferencesStorage = new Storage(
  { basePath: 'u', delimiter: ':', store: window.localStorage },
);

export type Mode = 'record' | 'replay' | 'mixed' | 'disabled';
const modes = new Set([ 'record', 'replay', 'mixed', 'disabled' ]);

const RR_MODE_KEY = 'mode';
interface State {
  mode: Mode;
}

export const rrState: State =
  { mode: rrConfigStorage.getWithDefault(RR_MODE_KEY, 'disabled') as Mode };

export const exportApiStorage = (): void => {
  rrConfigStorage.set('ui', userPreferencesStorage.toString());
  const fName = `det-export-${new Date().toLocaleString().replaceAll(' ', '')}.hal.json`;
  downloadText(
    fName,
    [ rrStorage.toString() ],
  );
};

export const setRRMode = (mode: Mode): void => {
  if (!modes.has(mode)) throw new Error(`unrecognized mode: ${mode}`);
  rrConfigStorage.set(RR_MODE_KEY, mode);
  // we set an auth token to the ui thinks it has already authenticated.
  if (mode === 'replay' && !globalStorage.authToken) globalStorage.authToken = 'fake-token';
  if (mode === 'replay') {
    const recordedUiConfig = rrConfigStorage.get<string>('ui');
    if (recordedUiConfig) {
      userPreferencesStorage.fromString(recordedUiConfig);
    }
  }
  rrState.mode = mode;
};

export const importApiStorageClipboard = async (): Promise<void> => {
  console.log('loading from clipboard..');
  const marshalled = await navigator.clipboard.readText();
  rrStorage.fromString(marshalled);
};

export const importApiStorageRemote = async (url: string): Promise<void> => {
  console.log('loading from remote..', url);
  const res = await window.fetch(url);
  const marshalled = await res.text();
  rrStorage.fromString(marshalled);
};

export const importApiStorage = async (): Promise<void> => {
  const contents = await requestTextFileUpload();
  console.log('loading from uploaded file');
  rrStorage.fromString(contents);
};

export const resetApiStorage = (): void => {
  rrStorage.reset();
};

// TODO move to browser/utils.
/*
  Provide a way to get file input from user.
*/
export const requestTextFileUpload = (): Promise<string> => {

  const el = document.createElement('input');
  el.setAttribute('type', 'file');

  return new Promise((resolve, reject) => {

    const readSingleFile = (e: Event) => {
      if (e.target === null) return;
      const files = (<HTMLInputElement>e.target).files;
      if (files === null) return;
      const file = files[0];
      const reader = new FileReader();
      reader.onload = function (e) {
        const contents = e.target && e.target.result;
        if (typeof contents === 'string') resolve(contents);
        document.body.removeChild(el);
        reject(contents);
      };
      reader.readAsText(file);
    };

    el.addEventListener('change', readSingleFile, false);
    document.body.insertBefore(el, document.body.firstChild);

  });
};

/*
  Generate a unique key used to represent a specific request given
  all the relevant parameters.
*/
export const genRequestKey = (name: string, params: unknown): string => {
  // name = name.replace('/', '-');
  const p = clone(params);
  const ineffectiveKeys = [ 'cancelToken', 'password', 'signal', 'username' ];
  ineffectiveKeys.forEach(k => delete p[k]);
  // WARN TODO we are relying on JSON.stringify to serialize objects with the same exact way between
  // different machines. If the assumption is broken there won't be a match.
  return `${name}:${sha512(JSON.stringify(p))}`;
};

const sanitizeResponse = (resp: any): any => {
  if (resp instanceof Object && 'data' in resp && 'token' in resp.data) {
    return { ...resp, token: 'omitted' };
  }
  return resp;
};

/*
  Given a unique key identifying the request and a reuqest fn decide when to
  record, replay, or pass through.
*/
export const applyRRMiddleware = async <T>(
  key: string,
  request: ()=> Eventually<T>,
): Promise<T> => {
  let response: T | null = null;
  switch (rrState.mode) {
    case 'disabled':
      return request();
    case 'record':
      response = await request();
      console.debug(`recording response to ${key} as `, response);
      rrResponseStorage.set(key, sanitizeResponse(response));
      return response;
    case 'replay':
      response = rrResponseStorage.get<T>(key);
      console.debug('replaying', key);
      if (response === null) throw handleError({
        message: 'Ran into an unrecorded request in replay mode.',
        silent: false,
        type: ErrorType.Ui,
      });
      return response;
    case 'mixed':
      response = rrResponseStorage.get<T>(key);
      if (response !== null) {
        console.debug('replaying', key);
        return response;
      } else {
        response = await request();
        console.debug(`recording response to ${key} as `, response);
        rrResponseStorage.set(key, sanitizeResponse(response));
        return response;
      }
    default:
      console.warn('unrecognized rr mode');
      return request();
  }
};

export const setupRRStorageAndReplay = (exportedContent: string): void => {
  userPreferencesStorage.reset();
  resetApiStorage();
  rrStorage.fromString(exportedContent);
  setRRMode('replay');
};

export const checkForImport = async (): Promise<void> => {
  const queryParamKey = 'import';
  const searchParams = new URLSearchParams(window.location.search);
  const remoteImport = searchParams.get(queryParamKey);
  if (!remoteImport) return;
  try {
    const res = await window.fetch(remoteImport);
    const marshalled = await res.text();
    setupRRStorageAndReplay(marshalled);
  } catch (e) {
    handleError({
      error: e,
      message: `failed to load from remote: ${remoteImport}`,
      silent: false,
      type: ErrorType.Ui,
    });
    console.error(e);
  }
};

export const devControls = {
  exportApiStorage,
  importApiStorage,
  importApiStorageClipboard,
  importApiStorageRemote,
  resetApiStorage,
  rrState: rrState,
  setRRMode,
};
