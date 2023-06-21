import React from 'react';

import { ValueOf } from 'types';

import css from './Label.module.scss';

export const LabelTypes = {
  TextOnly: 'textOnly',
} as const;

export type LabelTypes = ValueOf<typeof LabelTypes>;

interface Props extends React.HTMLAttributes<HTMLDivElement> {
  children: React.ReactNode;
  type?: LabelTypes;
}

const Label: React.FC<Props> = ({ className, children, type, ...props }: Props) => {
  const classes = [css.base];

  if (type) classes.push(css[type]);
  if (className) classes.push(className);

  return React.createElement('div', { className: classes.join(' '), ...props }, children);
};

export default Label;
