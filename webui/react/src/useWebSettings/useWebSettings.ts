import { atomFamily, selector, selectorFamily, SetterOrUpdater, useRecoilState } from 'recoil';

import { getUserWebSetting, updateUserWebSetting } from 'services/api';

import type {
  GlobalData,
  GlobalKeys,
  Param,
  ProjectDetailData,
  ProjectDetailKeys,
  UserWebSettings,
  UserWebSettingsDomainName,
  UserWebSettingsSubDomainName,
} from './useWebSettings.settings';
import { defaultUserWebSettings, UserWebSettingsDomain } from './useWebSettings.settings';

const allUserSettingsState = selector<UserWebSettings>({
  get: async () => {
    try {
      const response = await getUserWebSetting({});
      const resSettings = response.settings;
      const settings: UserWebSettings = { ...defaultUserWebSettings, ...resSettings };
      return settings;
    } catch (e) {
      throw new Error('unable to fetch userWebSettings data');
    }
  },
  key: 'allUserSettingsState',
});

const userSettingsDomainState = atomFamily<
  any,
  Param<UserWebSettingsDomainName, UserWebSettingsSubDomainName>
>({
  default: selectorFamily<unknown, Param<UserWebSettingsDomainName, UserWebSettingsSubDomainName>>({
    get:
      (param) =>
      ({ get }) => {
        const domains = get(allUserSettingsState);
        const domain = domains[param.domain];
        switch (param.domain) {
          case UserWebSettingsDomain.ProjectDetail:
            return (domain as ProjectDetailData)[param.subDomain as ProjectDetailKeys];
          case UserWebSettingsDomain.Global:
            return (domain as GlobalData)[param.subDomain as GlobalKeys];
          default:
            return (domain as ProjectDetailData)[param.subDomain as ProjectDetailKeys];
        }
      },
    key: 'userSettingsDomainQuery',
  }),
  effects: (param) => [
    ({ onSet }) => {
      onSet((newValue) => {
        updateUserWebSetting({
          setting: { value: { [param.domain]: { [param.subDomain]: newValue } } },
        });
      });
    },
  ],
  key: 'userSettingsDomainState',
});

const useWebSettings = <T extends keyof UserWebSettings, S extends keyof UserWebSettings[T]>(
  domain: T,
  subDomain: S,
): [UserWebSettings[T][S], SetterOrUpdater<UserWebSettings[T][S]>] => {
  const [userWebSettings, setUserWebSettings] = useRecoilState<UserWebSettings[T][S]>(
    userSettingsDomainState({
      domain: domain,
      subDomain: subDomain as UserWebSettingsSubDomainName,
    }),
  );

  return [userWebSettings, setUserWebSettings];
};

export default useWebSettings;
