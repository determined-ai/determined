import React from 'react';

import IconFilterButtons from './IconFilterButtons';

export default {
  component: IconFilterButtons,
  title: 'IconFilterButtons',
};

const defaultConfig = [
  { active: true, icon: 'experiment', id: 'experiment', label: 'Experiments' },
  { active: true, icon: 'notebook', id: 'notebook', label: 'Notebooks' },
  { active: false, icon: 'tensorboard', id: 'tensorboard', label: 'TensorBoards' },
  { active: false, icon: 'shell', id: 'shell', label: 'Shells' },
  { active: false, icon: 'command', id: 'command', label: 'Commands' },
];

export const Default = (): React.ReactNode => (
  <IconFilterButtons buttons={defaultConfig} />
);
