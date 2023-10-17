import React from 'react';

import Avatar, { Props as AvatarProps } from 'components/kit/Avatar';
import { User } from 'types';
import { getDisplayName } from 'utils/user';

export interface Props extends Omit<AvatarProps, 'darkLight' | 'text'> {
  user?: User;
}

const UserAvatar: React.FC<Props> = ({ user, ...rest }) => {
  const displayName = getDisplayName(user);

  return <Avatar {...rest} text={displayName} />;
};

export default UserAvatar;
