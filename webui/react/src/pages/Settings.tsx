import { Tabs } from 'antd';
import React, { useCallback, useState } from 'react';
import { useHistory, useParams } from 'react-router';

import Page from 'components/Page';
import SettingsAccount from 'pages/Settings/SettingsAccount';
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

const Settings: React.FC = () => {
  const { tab } = useParams<Params>();
  const [ tabKey, setTabKey ] = useState<TabType>(tab || DEFAULT_TAB_KEY);
  const basePath = paths.settings();
  const history = useHistory();

  const handleTabChange = useCallback(key => {
    setTabKey(key);
    history.replace(key === DEFAULT_TAB_KEY ? basePath : `${basePath}/${key}`);
  }, [ basePath, history ]);

  return (
    <Page
      bodyNoPadding
      id="cluster"
      stickyHeader
      title="Settings">
      <Tabs className="no-padding" defaultActiveKey={tabKey} onChange={handleTabChange}>
        <TabPane key="account" tab="Account">
          <SettingsAccount />
        </TabPane>
        <TabPane key="userManagement" tab="User Management">
          User Management
        </TabPane>
      </Tabs>
    </Page>
  );
};

export default Settings;
