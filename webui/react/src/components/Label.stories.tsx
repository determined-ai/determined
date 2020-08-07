import React from 'react';

import Label, { LabelTypes } from './Label';

export default {
  component: Label,
  title: 'Label',
};

export const Default = (): React.ReactNode => (
  <Label>Default Label</Label>
);

export const TextOnly = (): React.ReactNode => (
  <Label type={LabelTypes.TextOnly}>TextOnly Label</Label>
);
