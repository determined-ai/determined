import { useCallback } from 'react';

import { generateContext } from 'contexts';
import { getUsers } from 'services/api';
import { DetailedUser } from 'types';
import { isEqual } from 'utils/data';

const Users = generateContext<DetailedUser[] | undefined>({
  initialState: undefined,
  name: 'Users',
});

export const useFetchUsers = (canceler: AbortController): () => Promise<void> => {
  const users = Users.useStateContext();
  const setUsers = Users.useActionContext();

  return useCallback(async (): Promise<void> => {
    try {
      const usersResponse = await getUsers({ signal: canceler.signal });

      if (!isEqual(users, usersResponse)) {
        setUsers({ type: Users.ActionType.Set, value: usersResponse });
      }
    } catch (e) {}
  }, [ canceler, setUsers, users ]);
};

export default Users;
