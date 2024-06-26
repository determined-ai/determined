import { IntegrationType } from 'types';

export const createPachydermLineageLink = (
  integrationData: IntegrationType,
): string | undefined => {
  if (integrationData.pachyderm === undefined) return undefined;

  const { dataset, proxy } = integrationData.pachyderm;
  return `${proxy.scheme}://${proxy.host}:${proxy.port}/linage/${dataset.project}/repos/${dataset.repo}/commit/${dataset.commit}/?branchId=${dataset.branch}`;
};
