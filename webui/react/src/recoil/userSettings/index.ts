export type d = string
// import { atomFamily, selector, selectorFamily } from 'recoil';

// import { getUserWebSetting, updateUserWebSetting } from 'services/api';

// export enum ProjectDetailKey {
//   Archived = 'archived',
//   ColumnWidths = 'columnWidths',
//   Pinned = 'pinned',
//   SortKey = 'sortKey',
//   TableLimit = 'tableLimit',
//   UserFilter = 'userFilter',
// }

// export enum ThemeKey {
//   Theme = 'theme'
// }

// export type ProjectDetail = {
//   [ProjectDetailKey.Archived]: { archived: boolean };
//   [ProjectDetailKey.ColumnWidths]: {columnWidths: number[]};
//   [ProjectDetailKey.Pinned]: {pinned: Record<number, number[]>};
//   [ProjectDetailKey.SortKey]: {sortKey: string};
//   [ProjectDetailKey.TableLimit]: {tableLimit: number};
//   [ProjectDetailKey.UserFilter]: {userFilter: number[]};
// };

// export type Theme = {
//   [ThemeKey.Theme]: {theme: 'dark' | 'light' | 'system'};
// };

// export type AllData = ProjectDetail & Theme;

// const defaultAllData = {
//   [ProjectDetailKey.Archived]: { archived: false },
//   [ProjectDetailKey.ColumnWidths]: { columnWidths: [] },
//   [ProjectDetailKey.Pinned]: { pinned: { 1: [] } },
//   [ProjectDetailKey.SortKey]: { sortKey: '' },
//   [ProjectDetailKey.TableLimit]: { tableLimit: 20 },
//   [ProjectDetailKey.UserFilter]: { userFilter: [] },
//   [ThemeKey.Theme]: { theme: 'system' },
// };

// type DomainName = ProjectDetailKey | ThemeKey;

// const allUserSettings = selector<AllData>({
//   get: async () => {
//     try {
//       const response = await getUserWebSetting({});
//       const resSettings = response.settings;
//       Object.keys(resSettings).forEach((key: string) => {
//         resSettings[key] = { [key]: resSettings[key] };
//       });
//       const settings: AllData = { ...defaultAllData, ...resSettings };
//       return settings;
//     } catch (e) {
//       throw new Error('d');
//     }
//   },
//   key: 'allUserSettings',
// });

// export const userSettingsDomainState = atomFamily<any, DomainName>({
//   default: selectorFamily<unknown, DomainName>({
//     get: (domain) => ({ get }) => {
//       const domains = get(allUserSettings);
//       return domains[domain];
//     },
//     key: 'userSettingsDomainQuery',
//   }),
//   effects: [
//     ({ onSet }) => {
//       onSet((newValue) => {
//         updateUserWebSetting({ setting: { value: newValue } });
//       });
//     },
//   ],
//   key: 'userSettingsDomainState',
// });

// export const multipleUserSettingsDomainStates = selectorFamily<
//     {[k: string]: any}, ProjectDetailKey[]
//   >({
//     get: (propaties) => ({ get }) => {
//       const obj = Object.fromEntries(
//         propaties
//           .filter((key) => key in get(allUserSettings))
//           .map((key) => [ key, get(userSettingsDomainState(key)) ]),
//       );
//       return obj;
//     },
//     key: 'multipleUserSettingsDomainStates',
//     set: (field) => ({ set }, newValue) => {
//       for (const a of field) {
//         set(userSettingsDomainState(a), { [a]: newValue });
//       }
//     },
//   });
