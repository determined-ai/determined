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

type Palette = 'bright' | 'muted';

export interface Props {
  text: string;
  hideTooltip?: boolean;
  /** do not color the bg based on text */
  noColor?: boolean;
  size?: Size;
  square?: boolean;
  /** Palette to use for background and text colors. Defaults to 'bright'. */
  palette?: Palette;
  inactive?: boolean;
  tooltipText?: string;
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

export const getColor = (name = '', darkLight: DarkLight, palette?: Palette): string => {
  const hslColor = name ? hex2hsl(md5(name).substring(0, 6)) : hex2hsl('#808080');
  if (palette === 'muted') {
    return hsl2str({
      ...hslColor,
      l: darkLight === DarkLight.Dark ? 80 : 90,
      s: darkLight === DarkLight.Dark ? 40 : 77,
    });
  }
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
  palette = 'bright',
  tooltipText,
  inactive,
}) => {
  const { ui } = useUI();

  const style = {
    backgroundColor: noColor ? 'var(--theme-stage-strong)' : getColor(text, ui.darkLight, palette),
    color: noColor ? 'var(--theme-stage-on-strong)' : palette === 'bright' ? 'white' : 'black',
  };
  const classes = [css.base, css[size]];
  if (square) classes.push(css.square);
  if (inactive) classes.push(css.inactive);

  const avatar = (
    <div className={classes.join(' ')} id="avatar" style={style}>
      {getInitials(text)}
    </div>
  );

  return hideTooltip ? (
    avatar
  ) : (
    <Tooltip content={tooltipText ?? text} placement="right">
      {avatar}
    </Tooltip>
  );
};

export interface GroupProps extends Omit<Props, 'text'> {
  items: string[];
}
export const AvatarGroup: React.FC<GroupProps> = ({ items, ...rest }) => {
  return (
    <div className={css.group}>
      {items.map((item, idx) => (
        <Avatar key={idx} text={item} {...rest} />
      ))}
    </div>
  );
};

export default Avatar;
