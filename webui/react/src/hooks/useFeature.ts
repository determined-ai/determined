import queryString from 'query-string';

import { initInfo, useDeterminedInfo } from 'stores/determinedInfo';
import { DeterminedInfo } from 'types';
import { Loadable } from 'utils/loadable';

// Add new feature switches below using `|`
export type ValidFeature =
  | 'rbac'
  | 'mock_permissions_read'
  | 'trials_comparison'
  | 'mock_permissions_all'
  | 'dashboard';

const queryParams = queryString.parse(window.location.search);

interface FeatureHook {
  isOn: (feature: ValidFeature) => boolean;
}

const useFeature = (): FeatureHook => {
  const info = Loadable.getOrElse(initInfo, useDeterminedInfo());
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
