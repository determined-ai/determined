import { makeToast } from 'hew/Toast';

import { FinalAction } from 'omnibar/tree-extension/types';
import { routeToReactUrl } from 'utils/routes';
/** generates a handler that alerts when called */
export const alertAction =
  (msg: string): FinalAction =>
  () => {
    makeToast({ title: msg });
  };
export const visitAction = (url: string) => (): void => routeToReactUrl(url);
export const noOp = (): void => undefined;
export const parseIds = (input: string): number[] => input.split(',').map((i) => parseInt(i));
