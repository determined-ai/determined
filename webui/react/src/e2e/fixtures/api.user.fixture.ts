import streamConsumers from 'stream/consumers';

import _ from 'lodash';

import { randIdAlphanumeric, safeName } from 'e2e/utils/naming';
import {
  UsersApi,
  V1PatchUser,
  V1PostUserRequest,
  V1PostUserResponse,
  V1User,
} from 'services/api-ts-sdk/api';

import { apiFixture } from './api';

export class ApiUserFixture extends apiFixture(UsersApi) {
  new({ userProps = {}, usernamePrefix = 'test-user' } = {}): V1PostUserRequest {
    const defaults = {
      isHashed: false,
      password: randIdAlphanumeric({ length: 12 }),
      user: {
        active: true,
        admin: true,
        username: safeName(usernamePrefix),
      },
    };
    return {
      ...defaults,
      ...userProps,
    };
  }

  /**
   * Creates a user with the given parameters via the API.
   * @param {V1PostUserRequest} req the user request with the config for the new user.
   * See apiUser.newRandom() for the default config.
   * @returns {Promise<V1PostUserRequest>} Representation of the created user. The request is returned since the
   * password is not stored on the V1User object and it is not returned in the response. However the Request is a
   * strict superset of the Response, so no info is lost.
   */
  async createUser(req: V1PostUserRequest): Promise<V1PostUserResponse> {
    const userResp = await this.api.postUser(req, {}).catch(async function (error) {
      const respBody = await streamConsumers.text(error.body);
      throw new Error(
        `Create User Request failed. Status: ${error.status} Request: ${JSON.stringify(
          req,
        )} Response: ${respBody}`,
      );
    });
    return _.merge(req, userResp);
  }

  /**
   * Edits a user with the given parameters via the API.
   * @param {number} id - the ID of the user to modify.
   * @param {V1PatchUser} user - the user request with the config for the new user.
   * See apiUser.newRandom() for the default config.
   * @returns {Promise<V1User>} Representation of the modified user. Note that this
   * does not include some fields like password.
   */
  async patchUser(id: number, user: V1PatchUser): Promise<V1User> {
    const userResp = await this.api.patchUser(id, user).catch(async function (error) {
      const respBody = await streamConsumers.text(error.body);
      throw new Error(
        `Patch User Request failed. Status: ${error.status} Request: ${JSON.stringify(
          user,
        )} Response: ${respBody}`,
      );
    });
    return userResp.user;
  }
}
