import { Dropdown, Space } from 'antd';
import type { DropDownProps, MenuProps } from 'antd';
import { FilterValue, SorterResult, TablePaginationConfig } from 'antd/lib/table/interface';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import Badge, { BadgeType } from 'components/Badge';
import Button from 'components/kit/Button';
import Empty from 'components/kit/Empty';
import Icon from 'components/kit/Icon';
import { useModal } from 'components/kit/Modal';
import Page from 'components/Page';
import InteractiveTable, {
  ColumnDef,
  InteractiveTableSettings,
} from 'components/Table/InteractiveTable';
import SkeletonTable from 'components/Table/SkeletonTable';
import { defaultRowClassName, getFullPaginationConfig } from 'components/Table/Table';
import WebhookCreateModalComponent from 'components/WebhookCreateModal';
import WebhookDeleteModalComponent from 'components/WebhookDeleteModal';
import usePermissions from 'hooks/usePermissions';
import { UpdateSettings, useSettings } from 'hooks/useSettings';
import { getWebhooks, testWebhook } from 'services/api';
import { V1Trigger, V1TriggerType } from 'services/api-ts-sdk/api';
import usePolling from 'shared/hooks/usePolling';
import { ValueOf } from 'shared/types';
import { isEqual } from 'shared/utils/data';
import { ErrorType } from 'shared/utils/error';
import { alphaNumericSorter } from 'shared/utils/sort';
import { Webhook } from 'types';
import handleError from 'utils/error';

import css from './WebhookList.module.scss';
import settingsConfig, { DEFAULT_COLUMN_WIDTHS, Settings } from './WebhookList.settings';

const WebhooksView: React.FC = () => {
  const [webhooks, setWebhooks] = useState<Webhook[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [canceler] = useState(new AbortController());
  const [selectedWebhook, setSelectedWebhook] = useState<Webhook>();

  const pageRef = useRef<HTMLElement>(null);

  const { canEditWebhooks } = usePermissions();

  const WebhookCreateModal = useModal(WebhookCreateModalComponent);
  const WebhookDeleteModal = useModal(WebhookDeleteModalComponent);

  const { settings, updateSettings } = useSettings<Settings>(settingsConfig);

  const fetchWebhooks = useCallback(async () => {
    try {
      const webhooks = await getWebhooks({}, { signal: canceler.signal });
      setWebhooks((prev) => {
        if (isEqual(prev, webhooks)) return prev;
        return webhooks;
      });
    } catch (e) {
      handleError(e, {
        publicSubject: 'Unable to fetch webhooks.',
        silent: true,
        type: ErrorType.Api,
      });
    } finally {
      setIsLoading(false);
    }
  }, [canceler.signal]);

  const fetchAll = useCallback(async () => {
    await Promise.allSettled([fetchWebhooks()]);
  }, [fetchWebhooks]);

  usePolling(fetchAll, { rerunOnNewFn: true });

  /**
   * Get new webhooks based on changes to the pagination and sorter.
   */
  useEffect(() => {
    setIsLoading(true);
    fetchWebhooks();
  }, [fetchWebhooks]);

  const WebhookActionMenu = useCallback(
    (record: Webhook): DropDownProps['menu'] => {
      const MenuKey = {
        DeleteWebhook: 'delete-webhook',
        TestWebhook: 'test-webhook',
      } as const;

      const funcs = {
        [MenuKey.DeleteWebhook]: () => {
          setSelectedWebhook(record);
          WebhookDeleteModal.open();
        },
        [MenuKey.TestWebhook]: async () => {
          try {
            await testWebhook({ id: record.id });
          } catch (e) {
            handleError(e, {
              publicSubject: 'Webhook Request Failed',
              silent: false,
            });
          }
        },
      };

      const onItemClick: MenuProps['onClick'] = (e) => {
        funcs[e.key as ValueOf<typeof MenuKey>]();
      };

      const menuItems: MenuProps['items'] = [
        { key: MenuKey.TestWebhook, label: 'Test Webhook' },
        { danger: true, key: MenuKey.DeleteWebhook, label: 'Delete Webhook' },
      ];

      return { items: menuItems, onClick: onItemClick };
    },
    [WebhookDeleteModal],
  );

  const columns = useMemo(() => {
    const actionRenderer = (_: string, record: Webhook) => (
      <Dropdown menu={WebhookActionMenu(record)} trigger={['click']}>
        <Button icon={<Icon name="overflow-vertical" />} type="text" />
      </Dropdown>
    );

    const webhookTriggerRenderer = (triggers: V1Trigger[]) =>
      triggers.map((t) => {
        if (t.triggerType === V1TriggerType.EXPERIMENTSTATECHANGE) {
          return (
            <li className={css.listBadge} key={t.id}>
              <Badge state={t.condition.state} type={BadgeType.State} />
            </li>
          );
        }
      });

    return [
      {
        dataIndex: 'url',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['url'],
        key: 'url',
        sorter: (a: Webhook, b: Webhook): number => alphaNumericSorter(a.url, b.url),
        title: 'URL',
      },
      {
        dataIndex: 'triggers',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['triggers'],
        key: 'triggers',
        render: webhookTriggerRenderer,
        title: 'Experiment State Triggers',
      },
      {
        dataIndex: 'webhookType',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['webhookType'],
        key: 'webhookType',
        sorter: (a: Webhook, b: Webhook): number =>
          alphaNumericSorter(a.webhookType, b.webhookType),
        title: 'Type',
      },
      {
        align: 'right',
        className: 'fullCell',
        dataIndex: 'action',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['action'],
        fixed: 'right',
        key: 'action',
        render: actionRenderer,
        title: '',
        width: DEFAULT_COLUMN_WIDTHS['action'],
      },
    ] as ColumnDef<Webhook>[];
  }, [WebhookActionMenu]);

  const handleTableChange = useCallback(
    (
      tablePagination: TablePaginationConfig,
      tableFilters: Record<string, FilterValue | null>,
      tableSorter: SorterResult<Webhook> | SorterResult<Webhook>[],
    ) => {
      if (Array.isArray(tableSorter)) return;

      const { columnKey, order } = tableSorter as SorterResult<Webhook>;
      if (!columnKey || !columns.find((column) => column.key === columnKey)) return;

      const newSettings = {
        sortDesc: order === 'descend',
        sortKey: columnKey,
        tableLimit: tablePagination.pageSize,
        tableOffset: ((tablePagination.current ?? 1) - 1) * (tablePagination.pageSize ?? 0),
      };
      updateSettings(newSettings);
    },
    [columns, updateSettings],
  );

  useEffect(() => {
    return () => canceler.abort();
  }, [canceler]);

  return (
    <Page
      containerRef={pageRef}
      id="webhooks"
      options={
        <Space>
          {canEditWebhooks && <Button onClick={WebhookCreateModal.open}>New Webhook</Button>}
        </Space>
      }
      title="Webhooks">
      {webhooks.length === 0 && !isLoading ? (
        <Empty
          description="Call external services when experiments complete or throw errors."
          icon="webhooks"
          title="No Webhooks Registered"
        />
      ) : settings ? (
        <InteractiveTable
          columns={columns}
          containerRef={pageRef}
          dataSource={webhooks}
          loading={isLoading}
          pagination={getFullPaginationConfig(
            {
              limit: settings.tableLimit,
              offset: settings.tableOffset,
            },
            webhooks.length,
          )}
          rowClassName={defaultRowClassName({ clickable: false })}
          rowKey="id"
          settings={settings as InteractiveTableSettings}
          showSorterTooltip={false}
          size="small"
          updateSettings={updateSettings as UpdateSettings}
          onChange={handleTableChange}
        />
      ) : (
        <SkeletonTable columns={columns.length} />
      )}
      <WebhookCreateModal.Component />
      <WebhookDeleteModal.Component webhook={selectedWebhook} />
    </Page>
  );
};

export default WebhooksView;
