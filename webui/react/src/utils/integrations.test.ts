import { PachydermIntegrationDataType } from 'types';

import { createPachydermLineageLink } from './integrations';

export const mockIntegrationData: PachydermIntegrationDataType = {
  dataset: {
    branch: 'test_branch',
    commit: 'commit_example_123',
    project: 'test-project',
    repo: 'test-data',
    token: 'token_example_123',
  },
  pachd: {
    host: 'test_host',
    port: 123456,
  },
  proxy: {
    host: 'test_host',
    port: 12,
    scheme: 'http',
  },
};
const expectedResult = `${mockIntegrationData.proxy.scheme}://${mockIntegrationData.proxy.host}:${mockIntegrationData.proxy.port}/lineage/${mockIntegrationData.dataset.project}/repos/${mockIntegrationData.dataset.repo}/commit/${mockIntegrationData.dataset.commit}/?branchId=${mockIntegrationData.dataset.branch}`;

describe('Integrations', () => {
  describe('createPachydermLineageLink', () => {
    it('should return the link when passed pachyderm integration data', () => {
      const result = createPachydermLineageLink(mockIntegrationData);
      expect(result).toBe(expectedResult);
    });
  });
});
