import { Tabs } from 'antd';
import queryString from 'query-string';
import React, { useCallback, useState } from 'react';
import { useHistory, useParams } from 'react-router';
import { useLocation } from 'react-router-dom';

import Page from 'components/Page';
import SettingsAccount from 'pages/Settings/SettingsAccount';
import UserManagement from 'pages/Settings/UserManagement';
import { paths } from 'routes/utils';

const { TabPane } = Tabs;

enum TabType {
  Account = 'account',
  UserManagement = 'user-management',
}

interface Params {
  tab?: TabType;
}

const DEFAULT_TAB_KEY = TabType.Account;

const SettingsContent: React.FC = () => {
  const { tab } = useParams<Params>();
  const location = useLocation();
  const [ tabKey, setTabKey ] = useState<TabType>(tab || DEFAULT_TAB_KEY);
  const history = useHistory();

  const { rbac } = queryString.parse(location.search);

  const handleTabChange = useCallback((key) => {
    setTabKey(key);

    const basePath = paths.settings(key);
    const query = queryString.stringify({ rbac });
    const newPath = key === DEFAULT_TAB_KEY ? basePath : `${basePath}/${key}`;
    history.replace(`${newPath}?${query}`);
  }, [ history, rbac ]);

  return rbac ? (
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
