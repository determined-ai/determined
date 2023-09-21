import React from 'react';

import Nameplate from 'components/kit/Nameplate';
import UserAvatar from 'components/UserAvatar';
import { User } from 'types';

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
