import { useCallback } from 'react';

import { generateContext } from 'contexts';
import { RestApiState } from 'hooks/useRestApi';
import { getUsers } from 'services/api';
import { DetailedUser } from 'types';

const Users = generateContext<RestApiState<DetailedUser[]>>({
  initialState: {
    errorCount: 0,
    hasLoaded: false,
    isLoading: false,
  },
  name: 'Users',
});

export const useFetchUsers = (canceler: AbortController): () => Promise<void> => {
  const setUsers = Users.useActionContext();

  return useCallback(async (): Promise<void> => {
    try {
      const usersResponse = await getUsers({ signal: canceler.signal });
      setUsers({
        type: Users.ActionType.Set,
        value: {
          data: usersResponse,
          errorCount: 0,
          hasLoaded: true,
          isLoading: false,
        },
      });
    } catch (e) {}
  }, [ canceler, setUsers ]);
};

export default Users;
