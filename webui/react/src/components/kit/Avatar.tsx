import React from 'react';

import { hex2hsl, hsl2str } from 'components/kit/internal/functions';
import md5 from 'components/kit/internal/md5';
import { DarkLight, ValueOf } from 'components/kit/internal/types';
import useUI from 'components/kit/Theme';
import Tooltip from 'components/kit/Tooltip';

import css from './Avatar.module.scss';

export const Size = {
  ExtraLarge: 'extra-large',
  ExtraSmall: 'extra-small',
  Large: 'large',
  Medium: 'medium',
  Small: 'small',
} as const;

export type Size = ValueOf<typeof Size>;

export interface Props {
  text: string;
  hideTooltip?: boolean;
  /** do not color the bg based on text */
  noColor?: boolean;
  size?: Size;
  square?: boolean;
  textColor?: 'black' | 'white'
}

export const getInitials = (name = ''): string => {
  // Reduce the name to initials.
  const initials = name
    .split(/\s+/)
    .map((n) => n.charAt(0).toUpperCase())
    .join('');

  // If initials are long, just keep the first and the last.
  return initials.length > 2
    ? `${initials.charAt(0)}${initials.substring(initials.length - 1)}`
    : initials;
};

export const getColor = (name = '', darkLight: DarkLight): string => {
  const hslColor = name ? hex2hsl(md5(name).substring(0, 6)) : hex2hsl('#808080');
  return hsl2str({
    ...hslColor,
    l: darkLight === DarkLight.Dark ? 38 : 60,
  });
};

const Avatar: React.FC<Props> = ({
  text,
  hideTooltip,
  noColor,
  size = Size.Small,
  square,
  textColor = 'white',
}) => {
  const { ui } = useUI();

  const style = {
    backgroundColor: noColor ? 'var(--theme-stage-strong)' : getColor(text, ui.darkLight),
    color: noColor ? 'var(--theme-stage-on-strong)' : textColor,
  };
  const classes = [css.base, css[size]];
  if (square) classes.push(css.square);

  const avatar = (
    <div className={classes.join(' ')} id="avatar" style={style}>
      {getInitials(text)}
    </div>
  );

  return hideTooltip ? (
    avatar
  ) : (
    <Tooltip content={text} placement="right">
      {avatar}
    </Tooltip>
  );
};

export default Avatar;
