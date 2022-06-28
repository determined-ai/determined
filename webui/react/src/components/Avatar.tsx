import React, { useEffect, useState } from 'react';

import { useStore } from 'contexts/Store';
import { useFetchUsers } from 'hooks/useFetch';
import Avatar, { Props } from 'shared/components/Avatar/Avatar';
import { getDisplayName } from 'utils/user';

const UserAvatar: React.FC<{
  name?: string;
  // TODO: separate components for
  // 1) displaying an abbreviated string as an Avatar and
  // 2) finding user by userId in the store and displaying string Avatar or profile image
  userId?: number;
} & Omit<Props, 'displayName'>> = ({ name, userId, ...rest }) => {

  const [ displayName, setDisplayName ] = useState('');
  const { users } = useStore();
  const fetchUsers = useFetchUsers(new AbortController());

  useEffect(() => {
    if (!name && userId) {
      if (!users.length) {
        fetchUsers();
      }
      const user = users.find(user => user.id === userId);
      setDisplayName(getDisplayName(user));
    } else if (name) {
      setDisplayName(name);
    }
  }, [ fetchUsers, userId, name, users ]);

  return <Avatar {...rest} displayName={displayName} />;
};

export default UserAvatar;
