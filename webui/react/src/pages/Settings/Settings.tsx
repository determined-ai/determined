import { Tabs } from 'antd';
import React, { useCallback, useMemo, useState } from 'react';
import { useLocation, useNavigate, useParams } from 'react-router-dom-v5-compat';

import Page from 'components/Page';
import SettingsAccount from 'pages/Settings/SettingsAccount';
import UserManagement from 'pages/Settings/UserManagement';
import { paths } from 'routes/utils';

const { TabPane } = Tabs;

export enum TabType {
  Account = 'Account',
  UserManagement = 'User Management',
}

type Params = {
  tab?: TabType;
}

const TAB_KEYS = {
  [TabType.Account]: 'account',
  [TabType.UserManagement]: 'user-management',
};
const DEFAULT_TAB_KEY = TabType.Account;

const SettingsContent: React.FC = () => {
  const { tab } = useParams<Params>();
  const location = useLocation();
  const [ tabKey, setTabKey ] = useState<TabType>(tab || DEFAULT_TAB_KEY);
  const navigate = useNavigate();

  const rbacEnabled = location.pathname.search('rbac-enabled') > 0;

  const showTabs = useMemo(() => {
    // TODO: Enable tabs for admin once user management finishes.
    return rbacEnabled;
  }, [ rbacEnabled ]);

  const handleTabChange = useCallback((key) => {
    setTabKey(key);

    const basePath = paths.settings(key);
    navigate(`${basePath}/${rbacEnabled ? 'rbac-enabled' : ''}`, { replace: true });
  }, [ navigate, rbacEnabled ]);

  return showTabs ? (
    <Tabs className="no-padding" defaultActiveKey={tabKey} onChange={handleTabChange}>
      <TabPane key={TAB_KEYS[TabType.Account]} tab={TabType.Account}>
        <SettingsAccount />
      </TabPane>
      <TabPane key={TAB_KEYS[TabType.UserManagement]} tab={TabType.UserManagement}>
        <UserManagement />
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
