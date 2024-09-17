import streamConsumers from 'stream/consumers';

import _ from 'lodash';

import { expect } from 'e2e/fixtures/global-fixtures';
import { safeName } from 'e2e/utils/naming';
import {
  V1PostWorkspaceRequest,
  V1PostWorkspaceResponse,
  WorkspacesApi,
} from 'services/api-ts-sdk/api';

import { ApiAuthFixture } from './api.auth.fixture';

export class ApiWorkspaceFixture {
  readonly apiAuth: ApiAuthFixture;
  constructor(apiAuth: ApiAuthFixture) {
    this.apiAuth = apiAuth;
  }

  new({ workspaceProps = {}, workspacePrefix = 'test-workspace' } = {}): V1PostWorkspaceRequest {
    const defaults = {
      name: safeName(workspacePrefix),
    };
    return {
      ...defaults,
      ...workspaceProps,
    };
  }

  private static normalizeUrl(url: string): string {
    if (url.endsWith('/')) {
      return url.substring(0, url.length - 1);
    }
    return url;
  }

  private async startWorkspaceRequest(): Promise<WorkspacesApi> {
    return new WorkspacesApi(
      { apiKey: await this.apiAuth.getBearerToken() },
      ApiWorkspaceFixture.normalizeUrl(this.apiAuth.baseURL),
      fetch,
    );
  }

  /**
   * Creates a workspace with the given parameters via the API.
   * @param {V1PostWorkspaceRequest} req the workspace request with the config for the new workspace.
   * See apiWorkspace.newRandom() for the default config.
   * @returns {Promise<V1PostWorkspaceRequest>} Representation of the created workspace. The request is returned since the
   * password is not stored on the V1Workspace object and it is not returned in the response. However the Request is a
   * strict superset of the Response, so no info is lost.
   */
  async createWorkspace(req: V1PostWorkspaceRequest): Promise<V1PostWorkspaceResponse> {
    const apiAuth = this.apiAuth;
    const workspaceResp = await (await this.startWorkspaceRequest())
      .postWorkspace(req, {})
      .catch(async function (error) {
        const respBody = await streamConsumers.text(error.body);
        if (error.status === 401) {
          const token = apiAuth.getBearerToken();
          throw new Error(
            `Create Workspace Request failed. Status: ${error.status} Request: ${JSON.stringify(
              req,
            )} Token: ${token} Response: ${respBody}`,
          );
        }
        throw new Error(
          `Create Workspace Request failed. Status: ${error.status} Request: ${JSON.stringify(
            req,
          )} Response: ${respBody}`,
        );
      });
    return _.merge(req, workspaceResp);
  }

  /**
   *
   * @summary Delete a workspace.
   * @param {number} id The id of the workspace.
   */
  async deleteWorkspace(id: number): Promise<void> {
    await expect
      .poll(
        async () => {
          const workspaceResp = await (await this.startWorkspaceRequest())
            .deleteWorkspace(id)
            .catch(async function (error) {
              const respBody = await streamConsumers.text(error.body);
              if (error.status === 404) {
                return { completed: true };
              }
              throw new Error(
                `Delete Workspace Request failed. Status: ${error.status} Request: ${JSON.stringify(
                  id,
                )} Response: ${respBody}`,
              );
            });
          return workspaceResp.completed;
        },
        {
          message: `Delete Project Request failed ${JSON.stringify(id)}`,
          timeout: 15_000,
        },
      )
      .toBe(true);
  }
}
