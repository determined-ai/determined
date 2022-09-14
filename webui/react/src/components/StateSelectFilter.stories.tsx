import { ComponentStory, Meta } from '@storybook/react';
import React, { useState } from 'react';

import StateSelectFilter from './StateSelectFilter';

export default {
  component: StateSelectFilter,
  title: 'Determined/StateSelectFilter',
} as Meta<typeof StateSelectFilter>;

export const Default = (): React.ReactNode => <StateSelectFilter />;

export const Custom: ComponentStory<typeof StateSelectFilter> = (args) => {
  const [ currentValue, setCurrentValue ] = useState('');
  return (
    <StateSelectFilter
      {...args}
      value={currentValue}
      onChange={(newValue) => setCurrentValue(newValue as string)}
    />
  );
};

Custom.args = { showCommandStates: true, showExperimentStates: true };
