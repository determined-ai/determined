import React, { PropsWithChildren } from 'react';

import css from './Label.module.scss';

export enum LabelTypes {
  TextOnly = 'textOnly',
}

interface Props {
  type?: LabelTypes;
}

const Label: React.FC<Props> = ({ children, type }: PropsWithChildren<Props>) => {
  const classes = [ css.base ];

  if (type) classes.push(css[type]);

  return React.createElement('div', { className: classes.join(' ') }, children);
};

export default Label;
