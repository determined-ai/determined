import React from 'react';

import { User } from 'types';

import Nameplate from './kit/Nameplate';
import UserAvatar from './kit/UserAvatar';

export interface Props {
  compact?: boolean;
  hideAvatarTooltip?: boolean;
  user?: User;
}

const UserBadge: React.FC<Props> = ({ user, compact, hideAvatarTooltip }) => {
  return (
    <Nameplate
      alias={user?.displayName}
      compact={compact}
      icon={<UserAvatar hideTooltip={hideAvatarTooltip} user={user} />}
      name={user?.username ?? ''}
    />
  );
};

export default UserBadge;
