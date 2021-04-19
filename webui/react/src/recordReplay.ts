/* eslint-disable no-console */

import { downloadText } from 'utils/browser';
import { clone } from 'utils/data';
import { Storage } from 'utils/storage';
import { Eventually } from 'utils/types';

export const rrStorage = new Storage({
  basePath: 'recs',
  store: window.localStorage,
});

type Mode = 'record' | 'replay' | 'mixed' | 'disabled';
const modes = new Set([ 'record', 'replay', 'mixed', 'disabled' ]);

const RR_MODE_KEY = 'rrMode';
interface State {
  mode: Mode;
  storage: Storage;
}

export const rrState: State = {
  mode: window.localStorage.getItem(RR_MODE_KEY) as Mode || 'disabled',
  storage: rrStorage,
};

export const exportApiStorage = (): void => {
  // TODO gather and add on metadata.
  const fName = `det-export${new Date().toLocaleString().replaceAll(' ', '')}.hal.json`;
  downloadText(
    fName,
    [ rrStorage.toString() ],
  );
};

export const setRRMode = (mode: Mode): void => {
  // FIXME don't direclty work with local storage.
  if (!modes.has(mode)) throw new Error(`unrecognized mode: ${mode}`);
  window.localStorage.setItem(RR_MODE_KEY, mode);
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

export const importApiStorage = (): void => {
  requestTextFileUpload((marshalled: string | null) => {
    if (!marshalled) return;
    console.log('loading from uploaded file');
    rrStorage.fromString(marshalled);
  });
};

export const resetApiStorage = (): void => {
  rrStorage.reset();
};

// TODO move to browser/utils.
/*
  Provide a way to get file input from user.
*/
export const requestTextFileUpload = (callback: (c: string | null) => void): void => {

  // TODO use antd.Modal
  const el = document.createElement('input');
  el.setAttribute('type', 'file');

  const readSingleFile = (e: Event) => {
    if (e.target === null) return;
    const files = (<HTMLInputElement>e.target).files;
    if (files === null) return;
    const file = files[0];
    const reader = new FileReader();
    reader.onload = function (e) {
      const contents = e.target && e.target.result;
      if (!(contents instanceof ArrayBuffer)) callback(contents);
      document.body.removeChild(el);
    };
    reader.readAsText(file);
  };

  el.addEventListener('change', readSingleFile, false);
  document.body.insertBefore(el, document.body.firstChild);
};

async function hashText(text: string) {
  const encoder = new TextEncoder();
  const data = encoder.encode(text);
  const hashBuffer = await crypto.subtle.digest('SHA-256', data);
  const hashArray = Array.from(new Uint8Array(hashBuffer));
  const hashHex = hashArray.map(b => b.toString(16).padStart(2, '0')).join('');
  return hashHex;
}

/*
  Generate a unique key used to represent a specific request given
  all the relevant parameters.
*/
export const genRequestKey = (name: string, params: unknown): Promise<string> => {
  // name = name.replace('/', '-');
  const p = clone(params);
  const ineffectiveKeys = [ 'cancelToken', 'password', 'signal', 'username' ];
  ineffectiveKeys.forEach(k => delete p[k]);
  // WARN TODO we are relying on JSON.stringify to serialize objects with the same exact way between
  // different machines. If the assumption is broken there won't be a match.
  const key = name + '' + JSON.stringify(p);
  // return key;
  return hashText(key);
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
      console.log(`recording response to ${key} as `, response);
      rrStorage.set(key, response);
      return response;
    case 'replay':
      console.log('replaying', key);
      response = rrStorage.get<T>(key);
      if (response === null) throw new Error('missing response'); // TODO DaError
      return response;
    case 'mixed':
      response = rrStorage.get<T>(key);
      if (response !== null) {
        console.log('replaying', key);
        return response;
      } else {
        response = await request();
        console.log(`recording response to ${key} as `, response);
        rrStorage.set(key, response);
        return response;
      }
    default:
      console.warn('unrecognized rr mode');
      return request();
  }
};

export const checkForImport = async () => {
  const queryParamKey = 'import';
  const searchParams = new URLSearchParams(window.location.search);
  const remoteImport = searchParams.get(queryParamKey);
  if (!remoteImport) return;
  try {
    await importApiStorageRemote(remoteImport);
    setRRMode('replay');
  } catch (e) {
    console.error('failed to load from remote', remoteImport);
    console.error(e);
  }
};

export const devControls = {
  exportApiStorage,
  importApiStorage,
  importApiStorageRemote,
  resetApiStorage,
  rrState: rrState,
  setRRMode,
};
