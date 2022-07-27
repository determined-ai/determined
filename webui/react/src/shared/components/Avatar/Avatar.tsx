import { Tooltip } from 'antd';
import React from 'react';

import { DarkLight } from 'shared/themes';
import { ClassNameProp } from 'shared/types';
import { hex2hsl, hsl2str } from 'shared/utils/color';
import md5 from 'shared/utils/md5';

import css from './Avatar.module.scss';

export interface Props extends ClassNameProp {
  darkLight: DarkLight;
  displayName: string;
  hideTooltip?: boolean;
  large?: boolean;
  /** do not color the bg based on displayName */
  noColor?: boolean;
  square?: boolean;
}

const getInitials = (name = ''): string => {
  // Reduce the name to initials.
  const initials = name
    .split(/\s+/)
    .map((n) => n.charAt(0).toUpperCase())
    .join('');

  // If initials are long, just keep the first and the last.
  return initials.length > 2 ?
    `${initials.charAt(0)}${initials.substring(initials.length - 1)}` : initials;
};

const getColor = (name = '', darkLight: DarkLight): string => {
  const hslColor = name ? hex2hsl(md5(name).substring(0, 6)) : hex2hsl('#808080');
  return hsl2str({
    ...hslColor,
    l: darkLight === DarkLight.Dark ? 38 : 60,
  });
};

const Avatar: React.FC<Props> = (
  { noColor, square, className, hideTooltip, large, displayName, darkLight },
) => {
  const style = {
    backgroundColor: noColor ? 'var(--theme-stage-strong)' : getColor(displayName, darkLight),
    borderRadius: square ? '10%' : '100%',
  };
  const classes = [ css.base ];

  if (large) classes.push(css.large);
  if (className) classes.push(className);
  const avatar = (
    <div className={classes.join(' ')} id="avatar" style={style}>
      {getInitials(displayName)}
    </div>
  );

  return hideTooltip ? avatar : <Tooltip placement="right" title={displayName}>{avatar}</Tooltip>;
};

export default Avatar;
