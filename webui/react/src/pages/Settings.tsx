import { Tabs } from 'antd';
import queryString from 'query-string';
import React, { useCallback, useState } from 'react';
import { useLocation, useNavigate, useParams } from 'react-router-dom';

import Page from 'components/Page';
import SettingsAccount from 'pages/Settings/SettingsAccount';
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
  const basePath = paths.settings();
  const navigate = useNavigate();

  const { rbac } = queryString.parse(location.search);

  const handleTabChange = useCallback((key) => {
    setTabKey(key);
    navigate(key === DEFAULT_TAB_KEY ? basePath : `${basePath}/${key}`, { replace: true });
  }, [ basePath, navigate ]);

  return rbac ? (
    <Tabs className="no-padding" defaultActiveKey={tabKey} onChange={handleTabChange}>
      <TabPane key="account" tab="Account">
        <SettingsAccount />
      </TabPane>
      <TabPane key="userManagement" tab="User Management">
        User Management
      </TabPane>
    </Tabs>
  ) : <SettingsAccount />;
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
