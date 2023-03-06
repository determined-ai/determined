import React from 'react';

import { UserNameFields } from 'utils/user';

import Nameplate from './Nameplate';
import UserAvatar from './UserAvatar';

export interface Props {
  className?: string;
  compact?: boolean;
  user: UserNameFields;
}

const UserBadge: React.FC<Props> = ({ user, compact, className }) => {
  return (
    <Nameplate
      alias={user?.displayName}
      className={className}
      compact={compact}
      icon={<UserAvatar user={user} />}
      name={user?.username}
    />
  );
};

export default UserBadge;
