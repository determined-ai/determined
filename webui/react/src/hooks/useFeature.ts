import { useObservable } from 'micro-observables';

import determinedStore, { DeterminedInfo } from 'stores/determinedInfo';

// Add new feature switches below using `|`
export type ValidFeature =
  | 'rbac'
  | 'mock_permissions_read'
  | 'trials_comparison'
  | 'mock_permissions_all'
  | 'dashboard'
  | 'explist_v2'
  | 'chart';

const queryParams = new URLSearchParams(window.location.search);

interface FeatureHook {
  isOn: (feature: ValidFeature) => boolean;
}

const useFeature = (): FeatureHook => {
  const info = useObservable(determinedStore.info);
  return { isOn: (ValidFeature) => IsOn(ValidFeature, info) };
};

const IsOn = (feature: string, info: DeterminedInfo): boolean => {
  const { rbacEnabled, featureSwitches } = info;
  switch (feature) {
    case 'rbac':
      return rbacEnabled || queryParams.get(`f_${feature}`) === 'on';
    default:
      return queryParams.get(`f_${feature}`) === 'on' || featureSwitches.includes(feature);
  }
};

export default useFeature;
