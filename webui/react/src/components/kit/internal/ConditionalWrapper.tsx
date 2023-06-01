import React, { ReactElement } from 'react';

interface Props {
  children: ReactElement;
  condition: boolean;
  falseWrapper?: (children: ReactElement) => JSX.Element;
  wrapper: (children: ReactElement) => JSX.Element;
}

/*
 * Note: If the condition changes and children is a component,
 * the child component will go through an unmount and mount.
 */
export const ConditionalWrapper: React.FC<Props> = ({ condition, children, ...props }: Props) => {
  if (condition) return props.wrapper(children);
  if (!condition && props.falseWrapper) return props.falseWrapper(children);
  return children;
};
