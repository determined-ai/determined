import { message } from 'antd';

import { routeAll } from 'routes/utils';
export const alertAction = (msg: string) => ((): unknown => message.info(msg));
export const visitAction = (url: string) => ((): void => routeAll(url));
export const noOp = (): void => undefined;
export const parseIds = (input: string): number[] => input.split(',').map(i => parseInt(i));
