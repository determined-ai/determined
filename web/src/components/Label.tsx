import React, { PropsWithChildren } from 'react';

import css from './Label.module.scss';

export enum LabelTypes {
  TextOnly = 'textOnly',
}

interface Props extends React.HTMLAttributes<HTMLDivElement> {
  type?: LabelTypes;
}

const Label: React.FC<Props> = ({
  className,
  children,
  type,
  ...props
}: PropsWithChildren<Props>) => {
  const classes = [ css.base ];

  if (type) classes.push(css[type]);
  if (className) classes.push(className);

  return React.createElement('div', { className: classes.join(' '), ...props }, children);
};

export default Label;
