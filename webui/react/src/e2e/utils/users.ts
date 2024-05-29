import { V1PostUserRequest } from 'services/api-ts-sdk/api';

export const saveTestUser = (
  user: V1PostUserRequest,
  users: Map<number, V1PostUserRequest>,
): void => {
  if (user.user?.id === undefined) {
    throw new Error('User has an object but has no data.');
  }
  users.set(user.user.id, user);
};
