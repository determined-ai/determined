import React from 'react';

import Avatar, { Props as AvatarProps } from 'components/kit/Avatar';
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
      statusText={deactivated ? '(deactivated)' : undefined}
      text={displayName}
    />
  );
};

export default UserAvatar;
