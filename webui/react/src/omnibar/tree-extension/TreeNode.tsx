import * as React from 'react';

import { BaseNode } from 'omnibar/tree-extension/types';

import css from './TreeNode.module.scss';

type ResultRenderer<T> = (
  {
    item,
    isSelected,
    isHighlighted,
  }: {
    isHighlighted: boolean;
    isSelected: boolean;
    item: T;
  } & React.HTMLAttributes<HTMLElement>
) => JSX.Element;

/*
Renders a single option presented by the Tree Omnibar extention.
*/
const TreeNode: ResultRenderer<BaseNode> = (props) => {
  const { item, isSelected, isHighlighted, ...rest } = props;

  const classes = [ css.base ];

  if (isSelected) {
    classes.push(css.selected);
  }

  if (isHighlighted) {
    classes.push(css.highlighted);
  }

  const textualRepr = item.label || item.title;

  return (
    <li className={classes.join(' ')} {...rest} title={textualRepr}>
      {textualRepr}
    </li>
  );
};

export default TreeNode;
