import React from 'react';

import Avatar, { Props as AvatarProps } from 'components/kit/utils/components/Avatar';
import { User } from 'components/kit/utils/types';
import useUI from 'shared/contexts/stores/UI';
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
