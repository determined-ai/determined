import { Tabs } from 'antd';
import React, { useCallback, useMemo, useState } from 'react';
import { useLocation, useNavigate, useParams } from 'react-router-dom';

import Page from 'components/Page';
import SettingsAccount from 'pages/Settings/SettingsAccount';
import UserManagement from 'pages/Settings/UserManagement';
import { paths } from 'routes/utils';

const { TabPane } = Tabs;

enum TabType {
  Account = 'account',
  UserManagement = 'user-management',
}

type Params = {
  tab?: TabType;
}

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
      <TabPane key="account" tab="Account">
        <SettingsAccount />
      </TabPane>
      <TabPane key="user-management" tab="User Management">
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
