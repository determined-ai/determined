import { useObservable } from 'micro-observables';

import determinedStore, { DeterminedInfo } from 'stores/determinedInfo';

// Add new feature switches below using `|`
export type ValidFeature = 'trials_comparison' | 'dashboard' | 'explist_v2' | 'chart';

const queryParams = new URLSearchParams(window.location.search);

interface FeatureHook {
  isOn: (feature: ValidFeature) => boolean;
}

const useFeature = (): FeatureHook => {
  const info = useObservable(determinedStore.info);
  return { isOn: (ValidFeature) => IsOn(ValidFeature, info) };
};

const IsOn = (feature: string, info: DeterminedInfo): boolean => {
  const { featureSwitches } = info;
  return queryParams.get(`f_${feature}`) === 'on' || featureSwitches.includes(feature);
};

export default useFeature;
