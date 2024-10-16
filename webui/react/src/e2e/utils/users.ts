import { V1User } from 'services/api-ts-sdk/api';

export interface TestUser extends V1User {
  password?: string;
  // require:
  id: number;
}
