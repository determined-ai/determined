import type { TabsProps } from 'antd';
import React, { useCallback, useMemo, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';

import Pivot from 'components/kit/Pivot';
import Page from 'components/Page';
import usePermissions from 'hooks/usePermissions';
import GroupManagement from 'pages/Settings/GroupManagement';
import UserManagement from 'pages/Settings/UserManagement';
import { paths } from 'routes/utils';
import determinedStore from 'stores/determinedInfo';
import { ValueOf } from 'types';
import { useObservable } from 'utils/observable';

export const TabType = {
  GroupManagement: 'Groups',
  UserManagement: 'Users',
} as const;

export type TabType = ValueOf<typeof TabType>;

type Params = {
  tab?: TabType;
};

const TAB_KEYS = {
  [TabType.UserManagement]: 'user-management',
  [TabType.GroupManagement]: 'group-management',
} as const;
const DEFAULT_TAB_KEY = TabType.UserManagement;

const SettingsContent: React.FC = () => {
  const navigate = useNavigate();
  const { tab } = useParams<Params>();
  const [tabKey, setTabKey] = useState<TabType>(tab || DEFAULT_TAB_KEY);

  const { rbacEnabled } = useObservable(determinedStore.info);
  const { canAdministrateUsers } = usePermissions();

  const handleTabChange = useCallback(
    (key: string) => {
      setTabKey(key as TabType);
      navigate(paths.admin(key), { replace: true });
    },
    [navigate],
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
    <Pivot
      activeKey={tab}
      defaultActiveKey={tabKey}
      destroyInactiveTabPane
      items={tabItems}
      onChange={handleTabChange}
    />
  );
};

const Admin: React.FC = () => (
  <Page
    breadcrumb={[
      {
        breadcrumbName: 'Admin Settings',
        path: paths.admin(),
      },
    ]}
    id="admin"
    stickyHeader>
    <SettingsContent />
  </Page>
);

export default Admin;
