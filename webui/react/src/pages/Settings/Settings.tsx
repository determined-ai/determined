import { Tabs } from 'antd';
import type { TabsProps } from 'antd';
import React, { useCallback, useMemo, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';

import Page from 'components/Page';
import { InteractiveTableSettings } from 'components/Table/InteractiveTable';
import useFeature from 'hooks/useFeature';
import usePermissions from 'hooks/usePermissions';
import useSettings from 'hooks/useSettings';
import GroupManagement from 'pages/Settings/GroupManagement';
import SettingsAccount from 'pages/Settings/SettingsAccount';
import UserManagement from 'pages/Settings/UserManagement';
import { paths } from 'routes/utils';
import { ValueOf } from 'shared/types';

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
  const { updateSettings } = useSettings<InteractiveTableSettings>({
    settings: [],
    storagePath: '',
  });

  const rbacEnabled = useFeature().isOn('rbac');
  const { canViewUsers } = usePermissions();

  const handleTabChange = useCallback(
    (key: string) => {
      updateSettings({});
      setTabKey(key as TabType);
      navigate(paths.settings(key), { replace: true });
    },
    [navigate, updateSettings],
  );

  const tabItems: TabsProps['items'] = useMemo(() => {
    const items: TabsProps['items'] = [
      { children: <SettingsAccount />, key: TAB_KEYS[TabType.Account], label: TabType.Account },
    ];

    if (canViewUsers) {
      items.push({
        children: <UserManagement />,
        key: TAB_KEYS[TabType.UserManagement],
        label: TabType.UserManagement,
      });
    }

    if (rbacEnabled) {
      items.push({
        children: <GroupManagement />,
        key: TAB_KEYS[TabType.GroupManagement],
        label: TabType.GroupManagement,
      });
    }
    return items;
  }, [canViewUsers, rbacEnabled]);

  return (
    <Tabs
      activeKey={tab}
      className="no-padding"
      defaultActiveKey={tabKey}
      destroyInactiveTabPane
      items={tabItems}
      onChange={handleTabChange}
    />
  );
};

const Settings: React.FC = () => (
  <Page bodyNoPadding id="cluster" stickyHeader title="Settings">
    <SettingsContent />
  </Page>
);

export default Settings;
