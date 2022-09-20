import { Tabs } from 'antd';
import React, { useCallback, useState } from 'react';
import { useParams } from 'react-router';
import { useHistory } from 'react-router-dom';

import Page from 'components/Page';
import useFeature from 'hooks/useFeature';
import usePermissions from 'hooks/usePermissions';
import GroupManagement from 'pages/Settings/GroupManagement';
import SettingsAccount from 'pages/Settings/SettingsAccount';
import UserManagement from 'pages/Settings/UserManagement';
import { paths } from 'routes/utils';

const { TabPane } = Tabs;

export enum TabType {
  Account = 'Account',
  UserManagement = 'User Management',
  GroupManagement = 'Group Management',
}

interface Params {
  tab?: TabType;
}

const TAB_KEYS = {
  [TabType.Account]: 'account',
  [TabType.UserManagement]: 'user-management',
  [TabType.GroupManagement]: 'group-management',
};
const DEFAULT_TAB_KEY = TabType.Account;

const SettingsContent: React.FC = () => {
  const history = useHistory();
  const { tab } = useParams<Params>();
  const [tabKey, setTabKey] = useState<TabType>(tab || DEFAULT_TAB_KEY);

  const rbacEnabled = useFeature().isOn('rbac');
  const canViewUsers = usePermissions().canViewUsers();

  const handleTabChange = useCallback((key) => {
    setTabKey(key);
    history.replace(paths.settings(key));
  }, [history]);

  return (
    <Tabs
      className="no-padding"
      defaultActiveKey={tabKey}
      destroyInactiveTabPane
      onChange={handleTabChange}>
      <TabPane key={TAB_KEYS[TabType.Account]} tab={TabType.Account}>
        <SettingsAccount />
      </TabPane>
      {(rbacEnabled || canViewUsers) && (
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
