import { useObservable } from 'micro-observables';

import determinedStore, { DeterminedInfo } from 'stores/determinedInfo';

// Add new feature switches below using `|`
export type ValidFeature = 'dashboard' | 'explist_v2' | 'chart' | 'rp_binding';

const FeatureDefault: { [K in ValidFeature]: boolean } = {
  chart: false,
  dashboard: false,
  explist_v2: true,
  rp_binding: false,
};

const queryParams = new URLSearchParams(window.location.search);

interface FeatureHook {
  isOn: (feature: ValidFeature) => boolean;
}

const useFeature = (): FeatureHook => {
  const info = useObservable(determinedStore.info);
  return { isOn: (ValidFeature) => IsOn(ValidFeature, info) };
};

// Priority: Default state < config settings < user settings < url
const IsOn = (feature: ValidFeature, info: DeterminedInfo): boolean => {
  const { featureSwitches } = info;
  // Read from default state
  let isOn = FeatureDefault[feature];

  // Read from config settings
  featureSwitches.includes(feature) && (isOn = true);
  featureSwitches.includes(`-${feature}`) && (isOn = false);

  // Read from url
  queryParams.get(`f_${feature}`) === 'on' && (isOn = true);
  queryParams.get(`f_${feature}`) === 'off' && (isOn = false);

  return isOn;
};

export default useFeature;
