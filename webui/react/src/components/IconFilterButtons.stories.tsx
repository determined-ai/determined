import React from 'react';

import IconFilterButtons from './IconFilterButtons';

export default {
  component: IconFilterButtons,
  title: 'IconFilterButtons',
};

const defaultConfig = [
  { active: true, icon: 'experiment', id: 'experiment', tag: 'Experiments' },
  { active: true, icon: 'jupyter-lab', id: 'jupyter-lab', tag: 'JupyterLabs' },
  { active: false, icon: 'tensorboard', id: 'tensorboard', tag: 'TensorBoards' },
  { active: false, icon: 'shell', id: 'shell', tag: 'Shells' },
  { active: false, icon: 'command', id: 'command', tag: 'Commands' },
];

export const Default = (): React.ReactNode => (
  <IconFilterButtons buttons={defaultConfig} />
);
