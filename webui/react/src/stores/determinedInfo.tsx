import { Loadable, Loaded, NotLoaded } from 'hew/utils/loadable';

import { getInfo } from 'services/api';
import { ValueOf } from 'types';
import { deepObservable } from 'utils/observable';

import PollingStore from './polling';

export interface SsoProvider {
  name: string;
  ssoUrl: string;
  type: string;
}

export const BrandingType = {
  Determined: 'determined',
  HPE: 'hpe',
} as const;

export type BrandingType = ValueOf<typeof BrandingType>;

export interface DeterminedInfo {
  branding?: BrandingType;
  checked: boolean;
  clusterId: string;
  clusterName: string;
  externalLoginUri?: string;
  externalLogoutUri?: string;
  featureSwitches: string[];
  isTelemetryEnabled: boolean;
  masterId: string;
  rbacEnabled: boolean;
  ssoProviders?: SsoProvider[];
  userManagementEnabled: boolean;
  version: string;
}

export interface Telemetry {
  enabled: boolean;
  segmentKey?: string;
}

const initInfo: DeterminedInfo = {
  branding: undefined,
  checked: false,
  clusterId: '',
  clusterName: '',
  featureSwitches: [],
  isTelemetryEnabled: false,
  masterId: '',
  rbacEnabled: false,
  ssoProviders: [],
  userManagementEnabled: true,
  version: process.env.VERSION || '',
};

class DeterminedStore extends PollingStore {
  #info = deepObservable<Loadable<DeterminedInfo>>(NotLoaded);

  public readonly loadableInfo = this.#info.readOnly();

  public readonly info = this.#info.select((info) => Loadable.getOrElse(initInfo, info));

  public readonly isServerReachable = this.#info.select((info) => {
    return Loadable.match(info, {
      _: () => false,
      Loaded: (info) => !!info.clusterId,
    });
  });

  protected async poll() {
    const response = await getInfo({ signal: this.canceler?.signal });
    this.#info.set(Loaded({ ...response, checked: true }));
  }

  protected override pollCatch(): void {
    this.#info.update((prev) => {
      const info = Loadable.getOrElse(initInfo, prev);
      return Loaded({ ...info, checked: true });
    });
  }
}

export default new DeterminedStore();
