import { Tooltip } from 'antd';
import React, { useEffect, useState } from 'react';

import { useStore } from 'contexts/Store';
import { useFetchUsers } from 'hooks/useFetch';
import { hex2hsl, hsl2str } from 'utils/color';
import md5 from 'utils/md5';
import { getDisplayName } from 'utils/user';

import css from './Avatar.module.scss';

interface Props {
  hideTooltip?: boolean;
  id?: string;
  large?: boolean;
  name?: string;
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

const Avatar: React.FC<Props> = ({ hideTooltip, id, name, large }: Props) => {
  const [ value, setValue ] = useState('');
  const { users } = useStore();
  const fetchUsers = useFetchUsers(new AbortController());

  useEffect(() => {
    if (!name && id) {
      if (!users.length) {
        fetchUsers();
      }
      const user = users.find(user => user.username === id);
      setValue(getDisplayName(user));
    } else if (name) {
      setValue(name);
    }
  }, [ fetchUsers, id, name, users ]);

  const style = { backgroundColor: getColor(value) };
  const classes = [ css.base ];
  if (large) classes.push(css.large);
  const avatar = (
    <div className={classes.join(' ')} id="avatar" style={style}>
      {getInitials(value)}
    </div>
  );
  return hideTooltip ? avatar : <Tooltip placement="right" title={value}>{avatar}</Tooltip>;
};

export default Avatar;
