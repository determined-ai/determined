import { V1PostUserRequest } from 'services/api-ts-sdk/api';

export const saveTestUserId = (user: V1PostUserRequest, userIds: number[]): void => {
  if (user.user?.id === undefined) {
    throw new Error('User has an object but has no data.');
  }
  userIds.push(user.user.id);
};
