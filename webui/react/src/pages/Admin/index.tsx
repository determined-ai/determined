import { Tabs } from 'antd';
import type { TabsProps } from 'antd';
import React, { useCallback, useMemo, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';

import Page from 'components/Page';
import useFeature from 'hooks/useFeature';
import usePermissions from 'hooks/usePermissions';
import { useSettings } from 'hooks/useSettings';
import GroupManagement from 'pages/Settings/GroupManagement';
import UserManagement from 'pages/Settings/UserManagement';
import { paths } from 'routes/utils';
import { ValueOf } from 'shared/types';

export const TabType = {
  GroupManagement: 'Group Management',
  UserManagement: 'User Management',
} as const;

export type TabType = ValueOf<typeof TabType>;

type Params = {
  tab?: TabType;
};

const TAB_KEYS = {
  [TabType.UserManagement]: 'user-management',
  [TabType.GroupManagement]: 'group-management',
};
const DEFAULT_TAB_KEY = TabType.UserManagement;

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
      navigate(paths.admin(key), { replace: true });
    },
    [navigate, updateSettings],
  );

  const tabItems: TabsProps['items'] = useMemo(() => {
    const items: TabsProps['items'] = [];

    if (canAdministrateUsers) {
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
  }, [canAdministrateUsers, rbacEnabled]);

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

const Admin: React.FC = (p) => (
  <Page bodyNoPadding id="admin" stickyHeader title="Admin Settings">
    <SettingsContent {...p} />
  </Page>
);

export default Admin;
