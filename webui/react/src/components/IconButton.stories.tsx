import { text, withKnobs } from '@storybook/addon-knobs';
import React from 'react';

import IconButton from './IconButton';

export default {
  component: IconButton,
  decorators: [ withKnobs ],
  title: 'IconButton',
};

export const Default = (): React.ReactNode => <IconButton icon="checkmark" tag="Okay" />;
export const Custom = (): React.ReactNode => (
  <IconButton icon={text('Icon Name', 'experiment')} tag={text('Label', 'Experiment)')} />
);
