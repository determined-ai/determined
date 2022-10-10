import { Button, Dropdown, Menu, Space } from 'antd';
import type { MenuProps } from 'antd';
import { SorterResult } from 'antd/lib/table/interface';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';

// import FilterCounter from 'components/FilterCounter';
// import InlineEditor from 'components/InlineEditor';
import InteractiveTable, { ColumnDef, InteractiveTableSettings } from 'components/InteractiveTable';
// import Link from 'components/Link';
import Page from 'components/Page';
import { defaultRowClassName, getFullPaginationConfig } from 'components/Table';
import TagList from 'components/TagList';
import useModalWebhookCreate from 'hooks/useModal/Webhook/useModalWebhookCreate';
import useSettings, { UpdateSettings } from 'hooks/useSettings';
// import { paths } from 'routes/utils';
import { deleteWebhook, getWebhooks } from 'services/api';
import { V1Trigger, V1WebhookType } from 'services/api-ts-sdk/api';
import Icon from 'shared/components/Icon/Icon';
import usePolling from 'shared/hooks/usePolling';
import { isEqual } from 'shared/utils/data';
import { ErrorType } from 'shared/utils/error';
import { alphaNumericSorter } from 'shared/utils/sort';
import { Webhook } from 'types';
import handleError from 'utils/error';

import css from './Webhooks.module.scss';
import settingsConfig, { DEFAULT_COLUMN_WIDTHS, Settings } from './Webhooks.settings';

const WebhooksView: React.FC = () => {
  const [webhooks, setWebhooks] = useState<Webhook[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [canceler] = useState(new AbortController());
  const pageRef = useRef<HTMLElement>(null);

  const { contextHolder: modalWebhookCreateContextHolder, modalOpen: openWebhookCreate } =
    useModalWebhookCreate({});

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

  // const tableSearchIcon = useCallback(() => <Icon name="search" size="tiny" />, []);

  // const resetFilters = useCallback(() => {
  //   resetSettings([...filterKeys, 'tableOffset']);
  // }, [resetSettings]);

  const WebhookActionMenu = useCallback((record: Webhook) => {
    enum MenuKey {
      DELETE_WEBHOOK = 'delete-webhook',
    }

    const funcs = {
      [MenuKey.DELETE_WEBHOOK]: () => {
        deleteWebhook({ id: record.id });
      },
    };

    const onItemClick: MenuProps['onClick'] = (e) => {
      funcs[e.key as MenuKey]();
    };

    const menuItems: MenuProps['items'] = [
      { danger: true, key: MenuKey.DELETE_WEBHOOK, label: 'Delete Webhook' },
    ];

    return <Menu items={menuItems} onClick={onItemClick} />;
  }, []);

  const columns = useMemo(() => {
    const actionRenderer = (_: string, record: Webhook) => (
      <Dropdown overlay={() => WebhookActionMenu(record)} trigger={['click']}>
        <Button className={css.overflow} type="text">
          <Icon name="overflow-vertical" />
        </Button>
      </Dropdown>
    );

    const webhookTestRenderer = (_: string, record: Webhook) => (
      <Button className={css.overflow} type="text">
        Test
      </Button>
    );

    const webhookTriggerRenderer = (triggers: V1Trigger[]) => (
      <TagList
        compact
        disabled={true}
        tags={triggers.map((t) => `${t.triggerType}|${t.condition}`)}
      />
    );

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
        dataIndex: 'test',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['test'],
        key: 'test',
        render: webhookTestRenderer,
        title: 'Test',
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
  }, []);

  const handleTableChange = useCallback(
    (tablePagination, tableFilters, tableSorter) => {
      if (Array.isArray(tableSorter)) return;

      const { columnKey, order } = tableSorter as SorterResult<Webhook>;
      if (!columnKey || !columns.find((column) => column.key === columnKey)) return;

      const newSettings = {
        sortDesc: order === 'descend',
        sortKey: columnKey,
        tableLimit: tablePagination.pageSize,
        tableOffset: (tablePagination.current - 1) * tablePagination.pageSize,
      };
      // const shouldPush = settings.tableOffset !== newSettings.tableOffset;
      updateSettings(newSettings, true);
    },
    [columns, settings.tableOffset, updateSettings],
  );

  useEffect(() => {
    return () => canceler.abort();
  }, [canceler]);

  const showCreateWebhookModal = useCallback(() => openWebhookCreate(), [openWebhookCreate]);

  const WebhookActionDropdown = useCallback(
    ({ record, onVisibleChange, children }) => (
      <Dropdown
        overlay={() => WebhookActionMenu(record)}
        trigger={['contextMenu']}
        onVisibleChange={onVisibleChange}>
        {children}
      </Dropdown>
    ),
    [WebhookActionMenu],
  );

  return (
    <Page
      containerRef={pageRef}
      id="webhooks"
      options={
        <Space>
          <Button onClick={showCreateWebhookModal}>New Webhook</Button>
        </Space>
      }
      title="Webhooks">
      {webhooks.length === 0 && !isLoading ? (
        <div className={css.emptyBase}>
          <div className={css.icon}>
            <Icon name="inbox" size="mega" />
          </div>
          <h4>No Webhooks Registered</h4>
          <p className={css.description}>
            Call external services when experiments complete or throw errors.
          </p>
        </div>
      ) : (
        <InteractiveTable
          columns={columns}
          containerRef={pageRef}
          // ContextMenu={ModelActionDropdown}
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
          rowKey="name"
          settings={settings as InteractiveTableSettings}
          showSorterTooltip={false}
          size="small"
          updateSettings={updateSettings as UpdateSettings<InteractiveTableSettings>}
          onChange={handleTableChange}
        />
      )}
      {modalWebhookCreateContextHolder}
    </Page>
  );
};

export default WebhooksView;
