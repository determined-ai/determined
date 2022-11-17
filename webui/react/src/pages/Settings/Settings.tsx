import { Tabs } from 'antd';
import React, { useCallback, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';

import Page from 'components/Page';
import useFeature from 'hooks/useFeature';
import usePermissions from 'hooks/usePermissions';
import { useSettings } from 'hooks/useSettings';
import GroupManagement from 'pages/Settings/GroupManagement';
import SettingsAccount from 'pages/Settings/SettingsAccount';
import UserManagement from 'pages/Settings/UserManagement';
import { paths } from 'routes/utils';
import { ValueOf } from 'shared/types';

const { TabPane } = Tabs;

export const TabType = {
  Account: 'Account',
  GroupManagement: 'Group Management',
  UserManagement: 'User Management',
} as const;

export type TabType = ValueOf<typeof TabType>;

type Params = {
  tab?: TabType;
};

const TAB_KEYS = {
  [TabType.Account]: 'account',
  [TabType.UserManagement]: 'user-management',
  [TabType.GroupManagement]: 'group-management',
};
const DEFAULT_TAB_KEY = TabType.Account;

const SettingsContent: React.FC = () => {
  const navigate = useNavigate();
  const { tab } = useParams<Params>();
  const [tabKey, setTabKey] = useState<TabType>(tab || DEFAULT_TAB_KEY);
  const { updateSettings } = useSettings<object>({
    applicableRoutespace: '',
    settings: {},
    storagePath: '',
  });

  const rbacEnabled = useFeature().isOn('rbac');
  const { canAdministrateUsers } = usePermissions();

  const handleTabChange = useCallback(
    (key: string) => {
      updateSettings({});
      setTabKey(key as TabType);
      navigate(paths.settings(key), { replace: true });
    },
    [navigate, updateSettings],
  );

  return (
    <Tabs
      activeKey={tab}
      className="no-padding"
      defaultActiveKey={tabKey}
      destroyInactiveTabPane
      onChange={handleTabChange}>
      <TabPane key={TAB_KEYS[TabType.Account]} tab={TabType.Account}>
        <SettingsAccount />
      </TabPane>
      {canAdministrateUsers && (
        <TabPane key={TAB_KEYS[TabType.UserManagement]} tab={TabType.UserManagement}>
          <UserManagement />
        </TabPane>
      )}
      {rbacEnabled && (
        <TabPane key={TAB_KEYS[TabType.GroupManagement]} tab={TabType.GroupManagement}>
          <GroupManagement />
        </TabPane>
      )}
    </Tabs>
  );
};

const Settings: React.FC = () => (
  <Page bodyNoPadding id="cluster" stickyHeader title="Settings">
    <SettingsContent />
  </Page>
);

export default Settings;
