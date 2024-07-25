import { map } from 'fp-ts/lib/Record';
import { Loadable } from 'hew/utils/loadable';
import { boolean, null as ioNull, partial, TypeOf, union } from 'io-ts';
import { useObservable } from 'micro-observables';

import determinedStore, { DeterminedInfo } from 'stores/determinedInfo';
import userSettings from 'stores/userSettings';

// add new feature switches here
export type ValidFeature =
  | 'explist_v2'
  | 'rp_binding'
  | 'genai'
  | 'flat_runs'
  | 'streaming_updates'
  | 'task_templates';

type FeatureDescription = {
  friendlyName: string;
  description: string;
  defaultValue: boolean;
  noUserControl?: boolean;
};

export const FEATURES: Record<ValidFeature, FeatureDescription> = {
  explist_v2: {
    defaultValue: true,
    description: 'Enable improved experiment listing, filtering, and comparison',
    friendlyName: 'New Experiment List',
  },
  flat_runs: {
    defaultValue: true,
    description:
      'Presents all runs in a project in a single view, rather than grouped into experiments',
    friendlyName: 'Flat Runs View',
  },
  genai: {
    defaultValue: false,
    description: 'Enable links to Generative AI Studio',
    friendlyName: 'Generative AI Studio (genai)',
    noUserControl: true,
  },
  rp_binding: {
    defaultValue: true,
    description: 'Allow resource pools to be assigned to workspaces',
    friendlyName: 'Resource Pool Binding',
  },
  streaming_updates: {
    defaultValue: false,
    description: 'Allow streaming updates through websockets for better performance',
    friendlyName: 'Streaming Updates',
  },
  task_templates: {
    defaultValue: true,
    description: 'Manage tempaltes through WebUI',
    friendlyName: 'Manage Templates',
  },
} as const;
export const FEATURE_SETTINGS_PATH = 'global-features';

// had to dig into fp-ts to get a partial record for the settings config
export const FeatureSettingsConfig = partial(map(() => union([boolean, ioNull]))(FEATURES));
export type FeatureSettingsConfig = TypeOf<typeof FeatureSettingsConfig>;

const queryParams = new URLSearchParams(window.location.search);

interface FeatureHook {
  isOn: (feature: ValidFeature) => boolean;
}

const useFeature = (): FeatureHook => {
  const info = useObservable(determinedStore.info);
  const featureSettings = useObservable(
    userSettings
      .get(FeatureSettingsConfig, FEATURE_SETTINGS_PATH)
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
  let isOn: boolean = FEATURES[feature]?.defaultValue ?? false;

  // Read from config settings
  featureSwitches.includes(feature) && (isOn = true);
  featureSwitches.includes(`-${feature}`) && (isOn = false);

  if (FEATURES[feature]?.noUserControl) return isOn;

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
