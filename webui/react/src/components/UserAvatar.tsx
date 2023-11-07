import Avatar, { Props as AvatarProps } from 'hew/Avatar';
import React from 'react';

import { User } from 'types';
import { getDisplayName } from 'utils/user';

export interface Props extends Omit<AvatarProps, 'darkLight' | 'text'> {
  user?: User;
  deactivated?: boolean;
}

const UserAvatar: React.FC<Props> = ({ user, deactivated, ...rest }) => {
  const displayName = getDisplayName(user);

  return (
    <Avatar
      {...rest}
      inactive={deactivated}
      text={displayName}
      tooltipText={deactivated ? `${displayName} (deactivated)` : displayName}
    />
  );
};

export default UserAvatar;
