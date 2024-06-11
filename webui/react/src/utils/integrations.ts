import { PachdIntegrationType } from 'types';

export const createIntegrationLink = (integrationData: PachdIntegrationType): string => {
  const { dataset, proxy } = integrationData.pachyderm;
  return `${proxy.scheme}://${proxy.host}:${proxy.port}/linage/${dataset.project}/repos/${dataset.repo}/commit/${dataset.commit}/?branchId=${dataset.branch}`;
};
