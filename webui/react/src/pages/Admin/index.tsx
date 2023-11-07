import type { TabsProps } from 'antd';
import Pivot from 'hew/Pivot';
import { Loadable } from 'hew/utils/loadable';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';

import Page from 'components/Page';
import usePermissions from 'hooks/usePermissions';
import GroupManagement from 'pages/Admin/GroupManagement';
import UserManagement from 'pages/Admin/UserManagement';
import { paths } from 'routes/utils';
import { getGroups } from 'services/api';
import determinedStore from 'stores/determinedInfo';
import userStore from 'stores/users';
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
  const [totalGroup, setTotalGroup] = useState<number | undefined>(undefined);
  const { rbacEnabled } = useObservable(determinedStore.info);
  const { canAdministrateUsers } = usePermissions();
  const canceler = useRef(new AbortController());

  const loadableUsers = useObservable(userStore.getUsers());

  const getGroupTotal = useCallback(async () => {
    const response = await getGroups(
      {
        limit: 1,
        offset: 0,
      },
      { signal: canceler.current.signal },
    );
    setTotalGroup(response.pagination?.total);
  }, [canceler]);

  useEffect(() => {
    getGroupTotal();
  }, [getGroupTotal]);

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
      Loadable.match(loadableUsers, {
        _: () => null,
        Loaded: (users) => {
          items.push({
            children: <UserManagement />,
            key: TAB_KEYS[TabType.UserManagement],
            label: `${TabType.UserManagement} (${users.length})`,
          });
        },
      });
    }

    if (rbacEnabled) {
      items.push({
        children: <GroupManagement />,
        key: TAB_KEYS[TabType.GroupManagement],
        label: `${TabType.GroupManagement} ${totalGroup !== undefined ? `(${totalGroup})` : ''}`,
      });
    }

    return items;
  }, [canAdministrateUsers, rbacEnabled, totalGroup, loadableUsers]);

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
