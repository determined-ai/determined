import { Tooltip } from 'antd';
import React from 'react';

import { hex2hsl, hsl2str } from 'utils/color';
import md5 from 'utils/md5';

import css from './Avatar.module.scss';

interface Props {
  hideTooltip?: boolean;
  large?: boolean;
  name: string;
}

const getInitials = (name = ''): string => {
  // Reduce the name to initials.
  const initials = name
    .split(/\s+/)
    .map(n => n.charAt(0).toUpperCase())
    .join('');

  // If initials are long, just keep the first and the last.
  return initials.length > 2 ? `${initials.charAt(0)}${initials.substr(-1)}` : initials;
};

const getColor = (name = ''): string => {
  const hexColor = md5(name).substr(0, 6);
  const hslColor = hex2hsl(hexColor);
  return hsl2str({ ...hslColor, l: 50 });
};

const Avatar: React.FC<Props> = ({ hideTooltip, name, large }: Props) => {
  const style = { backgroundColor: getColor(name) };
  const classes = [ css.base ];
  if (large) classes.push(css.large);
  const avatar = (
    <div className={classes.join(' ')} id="avatar" style={style}>
      {getInitials(name)}
    </div>
  );
  return hideTooltip ? avatar : <Tooltip placement="right" title={name}>{avatar}</Tooltip>;
};

export default Avatar;
