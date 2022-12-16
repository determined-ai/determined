import React from 'react';

import Avatar, { Props as AvatarProps } from 'shared/components/Avatar';
import useUI from 'shared/contexts/stores/UI';
import { useUsers } from 'stores/users';
import { DetailedUser } from 'types';
import { Loadable } from 'utils/loadable';
import { getDisplayName } from 'utils/user';

import css from './UserAvatar.module.scss';
export interface Props extends Omit<AvatarProps, 'darkLight' | 'displayName'> {
  compact?: boolean;
  table?: boolean;
  user?: DetailedUser;
  userId?: number;
}

const UserAvatar: React.FC<Props> = ({ userId, table, user, compact, ...rest }) => {
  const users = Loadable.getOrElse([], useUsers()); // TODO: handle loading state
  const { ui } = useUI();
  const u = user ? user : users.find((user) => user.id === userId);
  const displayName = getDisplayName(u);

  const avatar = <Avatar {...rest} darkLight={ui.darkLight} displayName={displayName} />;
  if (!table || !u) return avatar;
  const classnames = [css.avartarCard];
  if (compact) classnames.push(css.compact);
  return (
    <div className={classnames.join(' ')}>
      {avatar}
      <div className={css.names}>
        {u.displayName && <span className={css.displayName}>{u.displayName}</span>}
        {<span>{u.username}</span>}
      </div>
    </div>
  );
};

export default UserAvatar;
