import { useObservable } from 'micro-observables';

import determinedStore, { DeterminedInfo } from 'stores/determinedInfo';

// Add new feature switches below using `|`
export type ValidFeature = 'trials_comparison' | 'dashboard' | 'explist_v2' | 'chart';

const queryParams = new URLSearchParams(window.location.search);

interface FeatureHook {
  isOn: (feature: ValidFeature) => boolean | undefined;
}

const useFeature = (): FeatureHook => {
  const info = useObservable(determinedStore.info);
  return { isOn: (ValidFeature) => IsOn(ValidFeature, info) };
};

// Priority: Default state < config settings < user settings < url
const IsOn = (feature: string, info: DeterminedInfo): boolean | undefined => {
  const { featureSwitches } = info;
  // Default state undefined
  let isOn: boolean | undefined = undefined;

  // Read from config settings
  featureSwitches.includes(feature) && (isOn = true);
  featureSwitches.includes(`-${feature}`) && (isOn = false);

  // Read from url
  queryParams.get(`f_${feature}`) === 'on' && (isOn = true);
  queryParams.get(`f_${feature}`) === 'off' && (isOn = false);

  return isOn;
};

export default useFeature;
