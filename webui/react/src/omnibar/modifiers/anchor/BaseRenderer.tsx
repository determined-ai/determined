import * as React from 'react';

import { BaseNode } from 'omnibar/AsyncTree';

// Based on omnibar/src/modifier/AnchorRenderer.tsx

export const COLORS = {
  BLACK: '#000',
  BLUE: '#00f',
  DARKGRAY: '#ddd',
  GRAY: '#eee',
  GREEN: '#0f0',
  RED: '#f00',
  WHITE: '#fff',
};

export const DEFAULT_HEIGHT = 50;

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

// TODO move to x.module.scss

const ITEM_STYLE: React.CSSProperties = {
  borderBottomWidth: 1,
  borderColor: COLORS.DARKGRAY,
  borderLeftWidth: 1,
  borderRightWidth: 1,
  borderStyle: 'solid',
  borderTopWidth: 0,
  boxSizing: 'border-box',
  color: COLORS.BLACK,
  display: 'block',
  fontSize: 24,
  height: DEFAULT_HEIGHT,
  lineHeight: `${DEFAULT_HEIGHT}px`,
  maxWidth: 'calc(max(50vw, 30rem))',
  overflowY: 'hidden',
  paddingLeft: 15,
  paddingRight: 15,
  textDecoration: 'none',
  wordBreak: 'break-word',
  wordWrap: 'break-word',
};

const BaseRenderer: ResultRenderer<BaseNode> = (props) => {
  const { item, isSelected, isHighlighted, style, ...rest } = props;

  const mergedStyle = { ...ITEM_STYLE, ...style };

  if (isSelected) {
    mergedStyle.backgroundColor = COLORS.GRAY;
  }

  if (isHighlighted) {
    mergedStyle.backgroundColor = COLORS.DARKGRAY;
  }

  const textualRepr = item.label || item.title;

  return (
    <li style={mergedStyle} {...rest} title={textualRepr}>
      {textualRepr}
    </li>
  );
};

export default BaseRenderer;
