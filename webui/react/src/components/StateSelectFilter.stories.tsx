import { boolean, withKnobs } from '@storybook/addon-knobs';
import React, { useState } from 'react';

import StateSelectFilter from './StateSelectFilter';

export default {
  component: StateSelectFilter,
  decorators: [ withKnobs ],
  title: 'StateSelectFilter',
};

export const Default = (): React.ReactNode => (
  <StateSelectFilter />
);

export const Custom = (): React.ReactNode => {
  const [ currentValue, setCurrentValue ] = useState('');

  return <StateSelectFilter
    showCommandStates={boolean('showCommandStates', true)}
    showExperimentStates={boolean('showExperimentStates', true)}
    value={currentValue}
    onChange={(newValue) => setCurrentValue(newValue as string)}
  />;
};
