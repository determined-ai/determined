import streamConsumers from 'stream/consumers';

import _ from 'lodash';

import { expect } from 'e2e/fixtures/global-fixtures';
import { safeName } from 'e2e/utils/naming';
import { ProjectsApi, V1PostProjectRequest, V1PostProjectResponse } from 'services/api-ts-sdk/api';

import { apiFixture } from './api';

export class ApiProjectFixture extends apiFixture(ProjectsApi) {
  new({ projectProps = {}, projectPrefix = 'test-project' } = {}): V1PostProjectRequest {
    const defaults = {
      name: safeName(projectPrefix),
      workspaceId: 0,
    };
    return {
      ...defaults,
      ...projectProps,
    };
  }

  /**
   * Creates a project with the given parameters via the API.
   * @param {number} workspaceId workspace id to create the project in.
   * @param {V1PostProjectRequest} req the project request with the config for the new project.
   * See apiProject.newRandom() for the default config.
   * @returns {Promise<V1PostProjectRequest>} Representation of the created project. The request is returned since the
   * password is not stored on the V1Project object and it is not returned in the response. However the Request is a
   * strict superset of the Response, so no info is lost.
   */
  async createProject(
    workspaceId: number,
    req: V1PostProjectRequest,
  ): Promise<V1PostProjectResponse> {
    const projectResp = await this.api
      .postProject(workspaceId, req, {})
      .catch(async function (error) {
        const respBody = await streamConsumers.text(error.body);
        throw new Error(
          `Create Project Request failed. Status: ${error.status} Request: ${JSON.stringify(
            req,
          )} Response: ${respBody}`,
        );
      });
    return _.merge(req, projectResp);
  }

  /**
   *
   * @summary Delete a project.
   * @param {number} id The id of the project.
   */
  async deleteProject(id: number): Promise<void> {
    try {
      await this.api.deleteProject(id);
    } catch (error) {
      if (error instanceof Response && error.body) {
        const bodyText = error.text();
        throw new Error(
          `Delete Project Request failed. Status: ${error.status} Request: ${JSON.stringify({
            id,
          })} Response: ${bodyText}`,
        );
      }
    }
    await expect
      .poll(
        () =>
          this.api
            .getProject(id)
            .then(() => false)
            .catch((res) => res.status === 404),
        {
          message: `Delete Project Request failed ${JSON.stringify(id)}`,
          timeout: 15_000,
        },
      )
      .toBe(true);
  }
}
