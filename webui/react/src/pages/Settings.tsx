import { Tabs } from 'antd';
import queryString from 'query-string';
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

  const rbacEnabled = queryString.parse(location.search).rbac !== undefined;

  const showTabs = useMemo(() => {
    // TODO: Enable tabs for admin once user management finishes.
    // return user?.isAdmin || rbacEnabled;
    return rbacEnabled;
  }, [ rbacEnabled ]);

  const handleTabChange = useCallback((key) => {
    setTabKey(key);

    const basePath = paths.settings(key);
    const query = rbacEnabled ? 'rbac' : '';
    const newPath = key === DEFAULT_TAB_KEY ? basePath : `${basePath}/${key}`;
    navigate(`${newPath}?${query}`, { replace: true });
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
