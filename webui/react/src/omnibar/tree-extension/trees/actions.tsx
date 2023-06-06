import { Modal as AntdModal } from 'antd';

import root from 'omnibar/tree-extension/trees';
import { FinalAction } from 'omnibar/tree-extension/types';
import { dfsStaticRoutes } from 'omnibar/tree-extension/utils';
import { message } from 'utils/dialogApi';
import { routeToReactUrl } from 'utils/routes';
/** generates a handler that alerts when called */
export const alertAction =
  (msg: string): FinalAction =>
  () => {
    message.info(msg);
  };
export const visitAction = (url: string) => (): void => routeToReactUrl(url);
export const noOp = (): void => undefined;
export const parseIds = (input: string): number[] => input.split(',').map((i) => parseInt(i));

export const displayHelp = (): void => {
  const commands = dfsStaticRoutes([], [], root)
    .map((path) => path.reduce((acc, cur) => `${acc} ${cur.title}`, ''))
    .map((addr) => addr.replace('root ', ''))
    .sort();

  const keymap = [
    '"Enter" to select and auto complete an option.',
    '"Tab", "Up", or "Down" arrow keys to cycle through suggestions.',
    '"Escape" to close the bar.',
  ];
  /**
   * TODO: look into converting into UI Kit Modal.
   * Currently `displayHelp` doesn't seem to run inside React,
   * hence will not have access to ModalContext.
   */
  AntdModal.info({
    content: (
      <>
        <p>Keyboard shortcuts:</p>
        <ul>
          {keymap.map((el, idx) => (
            <li key={idx}>{el}</li>
          ))}
        </ul>
        <p>Available commands:</p>
        <ul>
          {commands.map((el, idx) => (
            <li key={idx}>{el}</li>
          ))}
        </ul>
      </>
    ),
    style: { minWidth: '700px' },
    title: 'Help',
  });
};
