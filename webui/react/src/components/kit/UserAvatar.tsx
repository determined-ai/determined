import React from 'react';

import useUI from 'components/kit/contexts/UI';
import Avatar, { Props as AvatarProps } from 'components/kit/internal/Avatar';
import { User } from 'components/kit/internal/types';
import { getDisplayName } from 'utils/user';

export interface Props extends Omit<AvatarProps, 'darkLight' | 'displayName'> {
  user?: User;
}

const UserAvatar: React.FC<Props> = ({ user, ...rest }) => {
  const { ui } = useUI();
  const displayName = getDisplayName(user);

  return <Avatar {...rest} darkLight={ui.darkLight} displayName={displayName} />;
};

export default UserAvatar;
