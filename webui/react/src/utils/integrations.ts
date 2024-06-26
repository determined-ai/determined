import { IntegrationType } from 'types';

export const createIntegrationLink = (integrationData: IntegrationType): string => {
  if (integrationData.pachyderm === undefined) return ''; // only parsing pachyderm integrations for now...

  const { dataset, proxy } = integrationData.pachyderm;
  return `${proxy.scheme}://${proxy.host}:${proxy.port}/linage/${dataset.project}/repos/${dataset.repo}/commit/${dataset.commit}/?branchId=${dataset.branch}`;
};
