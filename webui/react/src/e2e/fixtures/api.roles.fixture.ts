import streamConsumers from 'stream/consumers';

import _ from 'lodash';

import { RBACApi, V1AssignRolesRequest, V1AssignRolesResponse } from 'services/api-ts-sdk/api';

import { ApiAuthFixture } from './api.auth.fixture';

export class ApiRoleFixture {
  readonly apiAuth: ApiAuthFixture;
  constructor(apiAuth: ApiAuthFixture) {
    this.apiAuth = apiAuth;
  }

  new({ roleProps = {} } = {}): V1AssignRolesRequest {
    const defaults = {};
    return {
      ...defaults,
      ...roleProps,
    };
  }

  private static normalizeUrl(url: string): string {
    if (url.endsWith('/')) {
      return url.substring(0, url.length - 1);
    }
    return url;
  }

  private async startRoleRequest(): Promise<RBACApi> {
    return new RBACApi(
      { apiKey: await this.apiAuth.getBearerToken() },
      ApiRoleFixture.normalizeUrl(this.apiAuth.baseURL),
      fetch,
    );
  }

  async createAssignment(req: V1AssignRolesRequest): Promise<V1AssignRolesResponse> {
    const roleResp = await (await this.startRoleRequest())
      .assignRoles(req, {})
      .catch(async function (error) {
        const respBody = await streamConsumers.text(error.body);
        throw new Error(
          `Create Assignment Failed: ${error.status} Request: ${JSON.stringify(
            req,
          )} Response: ${respBody}`,
        );
      });
    return _.merge(req, roleResp);
  }
}
