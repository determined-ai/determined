import { PachydermIntegrationDataType } from 'types';

export const createPachydermLineageLink = (
  pachydermIntegrationData: PachydermIntegrationDataType,
): string => {
  const { dataset, proxy } = pachydermIntegrationData;
  return `${proxy.scheme}://${proxy.host}:${proxy.port}/lineage/${dataset.project}/repos/${dataset.repo}/commit/${dataset.commit}/?branchId=${dataset.branch}`;
};
