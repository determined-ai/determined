import { map } from 'fp-ts/lib/Record';
import { boolean, null as ioNull, keyof, partial, TypeOf, union } from 'io-ts';
import { useObservable } from 'micro-observables';

import determinedStore, { DeterminedInfo } from 'stores/determinedInfo';
import userSettings from 'stores/userSettings';
import { Loadable } from 'utils/loadable';

// add new feature switches here
export const FEATURES = {
  chart: 'New Charts',
  explist_v2: 'New Experiment List',
  rp_binding: 'Resource Pool Binding',
} as const;
export const FEATURE_SETTINGS_PATH = 'global-features';

const ioValidFeature = keyof(FEATURES);
export type ValidFeature = TypeOf<typeof ioValidFeature>;

// had to dig into fp-ts to get a partial record for the settings config
export const featureSettingsConfig = partial(
  map(() => union([boolean, ioNull]))(ioValidFeature.keys),
);
export type FeatureSettingsConfig = TypeOf<typeof featureSettingsConfig>;

export const FeatureDefault: { [K in ValidFeature]: boolean } = {
  chart: false,
  explist_v2: false,
  rp_binding: false,
};

const queryParams = new URLSearchParams(window.location.search);

interface FeatureHook {
  isOn: (feature: ValidFeature) => boolean;
}

const useFeature = (): FeatureHook => {
  const info = useObservable(determinedStore.info);
  const featureSettings = useObservable(
    userSettings
      .get(featureSettingsConfig, FEATURE_SETTINGS_PATH)
      .select((loadable) => Loadable.getOrElse(null, loadable)),
  );
  return { isOn: (feature: ValidFeature) => IsOn(feature, info, featureSettings) };
};

// Priority: Default state < config settings < user settings < url
const IsOn = (
  feature: ValidFeature,
  info: DeterminedInfo,
  settings: FeatureSettingsConfig | null,
): boolean => {
  const { featureSwitches } = info;
  // Read from default state
  let isOn: boolean = FeatureDefault[feature];

  // Read from config settings
  featureSwitches.includes(feature) && (isOn = true);
  featureSwitches.includes(`-${feature}`) && (isOn = false);

  // Read from user settings
  if (settings && feature in settings) {
    const userValue = settings[feature];
    // checks are split up bc typescript doesn't combine the typeguards properly
    if (userValue !== undefined && userValue !== null) {
      isOn = userValue;
    }
  }

  // Read from url
  queryParams.get(`f_${feature}`) === 'on' && (isOn = true);
  queryParams.get(`f_${feature}`) === 'off' && (isOn = false);

  return isOn;
};

export default useFeature;
