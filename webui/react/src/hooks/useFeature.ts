import queryString from 'query-string';

import { useStore } from 'contexts/Store';
import { DeterminedInfo } from 'types';

// Add new feature switches below using `|`
export type ValidFeature =
  | 'rbac'
  | 'mock_workspace_members'
  | 'mock_permissions_read'
  | 'trials_comparison'
  | 'mock_permissions_all';

const queryParams = queryString.parse(window.location.search);

interface FeatureHook {
  isOn: (feature: ValidFeature) => boolean;
}

const useFeature = (): FeatureHook => {
  const { info } = useStore();
  return { isOn: (ValidFeature) => IsOn(ValidFeature, info) };
};

const IsOn = (feature: string, info: DeterminedInfo): boolean => {
  const { rbacEnabled, featureSwitches } = info;
  switch (feature) {
    case 'rbac':
      return rbacEnabled || queryParams[`f_${feature}`] === 'on';
    default:
      return queryParams[`f_${feature}`] === 'on' || featureSwitches.includes(feature);
  }
};

export default useFeature;
