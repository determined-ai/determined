import { detExecSync } from 'e2e/utils/detCLI';

let rbacEnabled: boolean;

const getRbacEnabled = (): boolean => {
  const masterInfo = detExecSync('master info');
  const regexp = /rbacEnabled:\s*(?<enabled>true|false)/;

  const { groups } = regexp.exec(masterInfo) || {};

  return groups?.enabled === 'true';
};

export const isRbacEnabled = (): boolean => {
  if (rbacEnabled === undefined) rbacEnabled = getRbacEnabled();

  return rbacEnabled;
};
