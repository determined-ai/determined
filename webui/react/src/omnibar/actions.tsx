import { message, Modal } from 'antd';
import React, { ReactNode } from 'react';

import { routeAll } from 'routes/utils';

import { dfsStaticRoutes } from './AsyncTree';
import root from './sampleTree';
export const alertAction = (msg: string) => (() => { message.info(msg); });
export const visitAction = (url: string) => ((): void => routeAll(url));
export const noOp = (): void => undefined;
export const parseIds = (input: string): number[] => input.split(',').map(i => parseInt(i));
export const displayModal = (title: string, content: ReactNode) => {
  const modal = Modal.info({});
  modal.update({
    content: <pre>{content}</pre>,
    style: { minWidth: '70rem' },
    title, // FIXME
  });
  return modal;
};

export const displayHelp = () => {
  const commands = dfsStaticRoutes([], [], root)
    .map(path => path.reduce((acc, cur) => `${acc} ${cur.title}`, ''))
    .map(addr => addr.replace('root ', ''))
    .sort();

  const keymap = [
    '"Enter" to select and auto complete an option.',
    '"Tab", "Up", or "Down" arrow keys to cycle through suggestions.',
    '"Escape" to close the bar.',
  ];
  displayModal(
    'Help', (
      <>
        <p>
        Keyboard shortcuts:
        </p>
        <ul>
          {keymap.map((el, idx) => <li key={idx}>{el}</li>)}
        </ul>
        <p>
        Available commands:
        </p>
        <ul>
          {commands.map((el, idx) => <li key={idx}>{el}</li>)}
        </ul>
      </>
    ),
  );
  // this could be full suggestions when query is empty.
};
