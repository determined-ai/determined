import { Tabs } from 'antd';
import React, { useCallback, useEffect, useMemo, useState } from 'react';
import { useHistory, useParams } from 'react-router';
import { useLocation } from 'react-router-dom';

import Page from 'components/Page';
import GroupManagement from 'pages/Settings/GroupManagement';
import SettingsAccount from 'pages/Settings/SettingsAccount';
import UserManagement from 'pages/Settings/UserManagement';
import { paths } from 'routes/utils';

const { TabPane } = Tabs;

export enum TabType {
  Account = 'Account',
  UserManagement = 'User Management',
  GroupManagement = 'Group Management'
}

interface Params {
  tab?: TabType;
}

const TAB_KEYS = {
  [TabType.Account]: 'account',
  [TabType.UserManagement]: 'user-management',
  [TabType.GroupManagement]: 'group-management',
};
const DEFAULT_TAB_KEY = TAB_KEYS[TabType.Account];

const SettingsContent: React.FC = () => {
  const { tab } = useParams<Params>();
  const location = useLocation();
  const [ tabKey, setTabKey ] = useState<string>(tab || DEFAULT_TAB_KEY);
  const history = useHistory();

  const rbacEnabled = location.pathname.search('rbac-enabled') > 0;

  const showTabs = useMemo(() => {
    // TODO: Enable tabs for admin once user management finishes.
    return rbacEnabled;
  }, [ rbacEnabled ]);

  const handleTabChange = useCallback((key) => {
    setTabKey(key);
  }, []);

  useEffect(() => {
    const basePath = paths.settings(tabKey);
    history.replace(`${basePath}/${rbacEnabled ? 'rbac-enabled' : ''}`);
  }, [ tabKey, history, rbacEnabled ]);

  return showTabs ? (
    <Tabs
      className="no-padding"
      defaultActiveKey={tabKey}
      destroyInactiveTabPane
      onChange={handleTabChange}>
      <TabPane key={TAB_KEYS[TabType.Account]} tab={TabType.Account}>
        <SettingsAccount />
      </TabPane>
      <TabPane key={TAB_KEYS[TabType.UserManagement]} tab={TabType.UserManagement}>
        <UserManagement />
      </TabPane>
      <TabPane key={TAB_KEYS[TabType.GroupManagement]} tab={TabType.GroupManagement}>
        <GroupManagement />
      </TabPane>
    </Tabs>
  ) : (
    <SettingsAccount />
  );
};

const Settings: React.FC = () => (
  <Page
    bodyNoPadding
    id="cluster"
    stickyHeader
    title="Settings">
    <SettingsContent />
  </Page>
);

export default Settings;
