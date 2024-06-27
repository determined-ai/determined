import { IntegrationType } from 'types';

import { createPachydermLineageLink } from './integrations';

export const mockIntegrationData: IntegrationType = {
  pachyderm: {
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
  },
};
const { pachyderm } = mockIntegrationData;
const expectedResult = `${pachyderm!.proxy.scheme}://${pachyderm!.proxy.host}:${pachyderm!.proxy.port}/linage/${pachyderm!.dataset.project}/repos/${pachyderm!.dataset.repo}/commit/${pachyderm!.dataset.commit}/?branchId=${pachyderm!.dataset.branch}`;

describe('Integrations', () => {
  describe('createPachydermLineageLink', () => {
    it('should return the link when passed pachyderm integration data', () => {
      const result = createPachydermLineageLink(mockIntegrationData);
      expect(result).toBe(expectedResult);
    });

    it('should return undefined when passed undefined pachyderm integration data', () => {
      const result = createPachydermLineageLink({});
      expect(result).toBe(undefined);
    });
  });
});
