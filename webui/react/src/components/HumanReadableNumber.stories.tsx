import { number, text, withKnobs } from '@storybook/addon-knobs';
import React from 'react';

import HumanReadableNumber from './HumanReadableNumber';

export default {
  component: HumanReadableNumber,
  decorators: [ withKnobs ],
  parameters: { layout: 'centered' },
  title: 'HumanReadableNumber',
};

export const Default = (): React.ReactNode => (
  <HumanReadableNumber num={1} />
);

export const NotANumber = (): React.ReactNode => (
  <HumanReadableNumber num={NaN} />
);

export const PositiveInfinity = (): React.ReactNode => (
  <HumanReadableNumber num={Number.POSITIVE_INFINITY} />
);

export const NegativeInfinity = (): React.ReactNode => (
  <HumanReadableNumber num={Number.NEGATIVE_INFINITY} />
);

export const Undefined = (): React.ReactNode => (
  <HumanReadableNumber num={undefined} />
);

export const Null = (): React.ReactNode => (
  <HumanReadableNumber num={null} />
);

export const Custom = (): React.ReactNode => (
  <HumanReadableNumber
    num={number('num', 5270)}
    precision={number('precision', 3)}
    tooltipPrefix={text('tooltipPrefix', '')}
  />
);
