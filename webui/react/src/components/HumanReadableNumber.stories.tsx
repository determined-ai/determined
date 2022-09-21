import { ComponentMeta, ComponentStory } from '@storybook/react';
import React from 'react';

import HumanReadableNumber from './HumanReadableNumber';

export default {
  component: HumanReadableNumber,
  parameters: { layout: 'centered' },
  title: 'Determined/HumanReadableNumber',
} as ComponentMeta<typeof HumanReadableNumber>;

export const Default = (): React.ReactNode => <HumanReadableNumber num={1} />;

export const NotANumber = (): React.ReactNode => <HumanReadableNumber num={NaN} />;

export const PositiveInfinity = (): React.ReactNode => (
  <HumanReadableNumber num={Number.POSITIVE_INFINITY} />
);

export const NegativeInfinity = (): React.ReactNode => (
  <HumanReadableNumber num={Number.NEGATIVE_INFINITY} />
);

export const Undefined = (): React.ReactNode => <HumanReadableNumber num={undefined} />;

export const Null = (): React.ReactNode => <HumanReadableNumber num={null} />;

export const Custom: ComponentStory<typeof HumanReadableNumber> = (args) => (
  <HumanReadableNumber {...args} />
);

Custom.args = {
  num: 5270,
  precision: 3,
};
