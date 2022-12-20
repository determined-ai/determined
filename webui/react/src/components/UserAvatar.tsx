import React from 'react';

import Avatar, { Props as AvatarProps } from 'shared/components/Avatar';
import useUI from 'shared/contexts/stores/UI';
import { useCurrentUsers } from 'stores/users';
import { Loadable } from 'utils/loadable';
import { getDisplayName } from 'utils/user';

export interface Props extends Omit<AvatarProps, 'darkLight' | 'displayName'> {
  userId?: number;
}

const UserAvatar: React.FC<Props> = ({ userId, ...rest }) => {
  const currentUser = Loadable.match(useCurrentUsers(), {
    Loaded: (cUser) => cUser,
    NotLoaded: () => undefined,
  });
  const { ui } = useUI();
  const displayName = getDisplayName(currentUser);

  return <Avatar {...rest} darkLight={ui.darkLight} displayName={displayName} />;
};

export default UserAvatar;
