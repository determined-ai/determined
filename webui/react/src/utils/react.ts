import React, { ReactElement } from 'react';

type RN = ReactElement;

interface Props {
  condition: boolean;
  wrapper: (c: RN) => RN;
  children: RN;
}

export const ConditionalWrapper: React.FC<Props> = ({ condition, wrapper, children }: Props) => {
  return condition ? wrapper(children) : children;
};
