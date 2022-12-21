import React from 'react';

import Avatar, { Props as AvatarProps } from 'shared/components/Avatar';
import useUI from 'shared/contexts/stores/UI';
import { getDisplayName, UserNameFields } from 'utils/user';

import css from './UserBadge.module.scss';

export interface Props extends Omit<AvatarProps, 'darkLight' | 'displayName'> {
  compact?: boolean;
  user: UserNameFields;
}

const UserBadge: React.FC<Props> = ({ user, compact, ...rest }) => {
  const { ui } = useUI();
  const displayName = getDisplayName(user);

  const avatar = <Avatar {...rest} darkLight={ui.darkLight} displayName={displayName} />;
  const classnames = [css.avatarCard];
  if (compact) classnames.push(css.compact);

  return (
    <div className={classnames.join(' ')}>
      {avatar}
      <div className={css.names}>
        {user.displayName && <span className={css.displayName}>{user.displayName}</span>}
        {<span>{user.username}</span>}
      </div>
    </div>
  );
};

export default UserBadge;
