import { Modal } from 'hew/Modal';
import React from 'react';

import root from 'omnibar/tree-extension/trees';
import { dfsStaticRoutes } from 'omnibar/tree-extension/utils';

const HelpModalComponent: React.FC = () => {
  const commands = dfsStaticRoutes([], [], root)
    .map((path) => path.reduce((acc, cur) => `${acc} ${cur.title}`, ''))
    .map((addr) => addr.replace('root ', ''))
    .sort();

  const keymap = [
    '"Enter" to select and auto complete an option.',
    '"Tab", "Up", or "Down" arrow keys to cycle through suggestions.',
    '"Escape" to close the bar.',
  ];

  return (
    <Modal title="Help">
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
    </Modal>
  );
};

export default HelpModalComponent;
