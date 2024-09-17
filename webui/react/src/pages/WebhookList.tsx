import {
  FilterDropdownProps,
  FilterValue,
  SorterResult,
  TablePaginationConfig,
} from 'antd/lib/table/interface';
import Button from 'hew/Button';
import Dropdown, { MenuItem } from 'hew/Dropdown';
import Icon from 'hew/Icon';
import Message from 'hew/Message';
import { useModal } from 'hew/Modal';
import Row from 'hew/Row';
import { useToast } from 'hew/Toast';
import _ from 'lodash';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import Badge, { BadgeType } from 'components/Badge';
import Page from 'components/Page';
import InteractiveTable, { ColumnDef } from 'components/Table/InteractiveTable';
import SkeletonTable from 'components/Table/SkeletonTable';
import {
  defaultRowClassName,
  getFullPaginationConfig,
  taskWorkspaceRenderer,
} from 'components/Table/Table';
import TableFilterDropdown from 'components/Table/TableFilterDropdown';
import WebhookCreateModalComponent from 'components/WebhookCreateModal';
import WebhookDeleteModalComponent from 'components/WebhookDeleteModal';
import WebhookEditModalComponent from 'components/WebhookEditModal';
import WorkspaceFilter from 'components/WorkspaceFilter';
import useFeature from 'hooks/useFeature';
import usePermissions from 'hooks/usePermissions';
import usePolling from 'hooks/usePolling';
import { useSettings } from 'hooks/useSettings';
import { paths } from 'routes/utils';
import { getWebhooks, testWebhook } from 'services/api';
import { V1Trigger, V1TriggerType } from 'services/api-ts-sdk/api';
import workspaceStore from 'stores/workspaces';
import { Webhook } from 'types';
import { copyToClipboard } from 'utils/dom';
import handleError, { ErrorType } from 'utils/error';
import { useObservable } from 'utils/observable';
import { alphaNumericSorter } from 'utils/sort';

import css from './WebhookList.module.scss';
import settingsConfig, { DEFAULT_COLUMN_WIDTHS, Settings } from './WebhookList.settings';

const MenuKey = {
  CopyName: 'copy-name',
  DeleteWebhook: 'delete-webhook',
  EditWebhook: 'edit-webhook',
  TestWebhook: 'test-webhook',
} as const;

const WebhooksView: React.FC = () => {
  const [webhooks, setWebhooks] = useState<Webhook[]>([]);
  const [filteredWebhooks, setFilteredWebhooks] = useState<Webhook[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [canceler] = useState(new AbortController());
  const [selectedWebhook, setSelectedWebhook] = useState<Webhook>();
  const f_flat_runs = useFeature().isOn('flat_runs');
  const f_webhook = useFeature().isOn('webhook_improvement');
  const pageRef = useRef<HTMLElement>(null);

  const { canCreateWebhooks, canEditWebhooks } = usePermissions();

  const workspaces = useObservable(workspaceStore.workspaces).getOrElse([]);

  const WebhookCreateModal = useModal(WebhookCreateModalComponent);
  const WebhookDeleteModal = useModal(WebhookDeleteModalComponent);
  const WebhookEditModal = useModal(WebhookEditModalComponent);

  const { settings, updateSettings } = useSettings<Settings>(settingsConfig);

  const { openToast } = useToast();

  const fetchWebhooks = useCallback(async () => {
    try {
      const webhooks = await getWebhooks({}, { signal: canceler.signal });
      setWebhooks((prev) => {
        if (_.isEqual(prev, webhooks)) return prev;
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

  usePolling(fetchWebhooks, { rerunOnNewFn: true });

  /**
   * Get new webhooks based on changes to the pagination and sorter.
   */
  useEffect(() => {
    setIsLoading(true);
    fetchWebhooks();
  }, [fetchWebhooks]);

  useEffect(() => {
    if (Array.isArray(settings.workspaceIds)) {
      const idSet = new Set(settings.workspaceIds);
      setFilteredWebhooks(webhooks.filter((w) => idSet.has(w.workspaceId)));
    } else {
      setFilteredWebhooks(webhooks);
    }
  }, [webhooks, settings.workspaceIds]);

  const handleDropdown = useCallback(
    async (key: string, record: Webhook) => {
      switch (key) {
        case MenuKey.CopyName:
          try {
            await copyToClipboard(record.name);
            openToast({
              description: 'Webhook name copied to the clipboard.',
              severity: 'Confirm',
              title: 'Webhook Name Copied',
            });
          } catch (e) {
            openToast({
              description: (e as Error)?.message,
              severity: 'Warning',
              title: 'Unable to Copy to Clipboard',
            });
          }
          break;
        case MenuKey.DeleteWebhook:
          setSelectedWebhook(record);
          WebhookDeleteModal.open();
          break;
        case MenuKey.TestWebhook:
          try {
            await testWebhook({ id: record.id });
          } catch (e) {
            handleError(e, {
              publicSubject: 'Webhook Request Failed',
              silent: false,
            });
          }
          break;
        case MenuKey.EditWebhook:
          setSelectedWebhook(record);
          WebhookEditModal.open();
          break;
      }
    },
    [WebhookDeleteModal, WebhookEditModal, openToast],
  );

  const handleWorkspaceFilterApply = useCallback(
    (workspaces: string[]) => {
      updateSettings({
        tableOffset: 0,
        workspaceIds:
          workspaces.length !== 0 ? workspaces.map((workspace) => Number(workspace)) : undefined,
      });
    },
    [updateSettings],
  );

  const handleWorkspaceFilterReset = useCallback(() => {
    updateSettings({ tableOffset: 0, workspaceIds: undefined });
  }, [updateSettings]);

  const workspaceFilterDropdown = useCallback(
    (filterProps: FilterDropdownProps) => (
      <TableFilterDropdown
        {...filterProps}
        multiple
        values={settings.workspaceIds?.map((ws) => ws.toString())}
        width={220}
        onFilter={handleWorkspaceFilterApply}
        onReset={handleWorkspaceFilterReset}
      />
    ),
    [handleWorkspaceFilterApply, handleWorkspaceFilterReset, settings.workspaceIds],
  );

  const columns = useMemo(() => {
    const renderMenu: (record: Webhook) => MenuItem[] = (record: Webhook) => {
      const DROPDOWN_MENU = [];
      record.name && DROPDOWN_MENU.push({ key: MenuKey.CopyName, label: 'Copy Webhook Name' });
      canEditWebhooks(workspaces, record) &&
        DROPDOWN_MENU.push(
          { key: MenuKey.TestWebhook, label: 'Test Webhook' },
          { key: MenuKey.EditWebhook, label: 'Edit Webhook' },
          { danger: true, key: MenuKey.DeleteWebhook, label: 'Delete Webhook' },
        );
      return DROPDOWN_MENU;
    };
    const actionRenderer = (_: string, record: Webhook) => {
      const menu = renderMenu(record);
      return menu.length > 0 ? (
        <Dropdown menu={menu} onClick={(key) => handleDropdown(key, record)}>
          <Button icon={<Icon name="overflow-vertical" title="Action menu" />} type="text" />
        </Dropdown>
      ) : null;
    };

    const webhookTriggerRenderer = (triggers: V1Trigger[]) =>
      triggers.map((t) => {
        if (t.triggerType === V1TriggerType.EXPERIMENTSTATECHANGE) {
          return (
            <li className={css.listBadge} key={t.id}>
              <Badge state={t.condition.state} type={BadgeType.State} />
            </li>
          );
        }
        if (t.triggerType === V1TriggerType.TASKLOG) {
          return (
            <li className={css.listBadge} key={t.id}>
              <Badge>TASKLOG</Badge>
            </li>
          );
        }
        if (t.triggerType === V1TriggerType.CUSTOM) {
          return (
            <li className={css.listBadge} key={t.id}>
              <Badge>CUSTOM</Badge>
            </li>
          );
        }
        return <></>;
      });

    const columns = [
      {
        dataIndex: 'name',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['name'],
        key: 'name',
        sorter: (a: Webhook, b: Webhook): number => alphaNumericSorter(a.name, b.name),
        title: 'Name',
      },
      {
        align: 'center',
        dataIndex: 'workspaceIds',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['workspaceIds'],
        filterDropdown: workspaceFilterDropdown,
        filters: workspaces.map((ws) => ({
          text: <WorkspaceFilter workspace={ws} />,
          value: ws.id,
        })),
        isFiltered: (settings: Settings) => !!settings.workspaceIds?.length,
        key: 'workspaceId',
        render: (_v: string, record: Webhook) => taskWorkspaceRenderer(record, workspaces),
        title: 'Workspace',
      },
      {
        dataIndex: 'mode',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['mode'],
        key: 'mode',
        render: (m: string, record: Webhook) => (record.workspaceId > 0 ? m : 'Global'),
        title: 'Mode',
      },
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
        title: `${f_flat_runs ? 'Search' : 'Experiment'} State Triggers`,
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

    if (!f_webhook) {
      return columns.slice(3);
    }

    return columns;
  }, [
    f_flat_runs,
    handleDropdown,
    workspaces,
    workspaceFilterDropdown,
    f_webhook,
    canEditWebhooks,
  ]);

  const handleTableChange = useCallback(
    (
      tablePagination: TablePaginationConfig,
      _tableFilters: Record<string, FilterValue | null>,
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
      breadcrumb={[
        {
          breadcrumbName: 'Webhooks',
          path: paths.webhooks(),
        },
      ]}
      containerRef={pageRef}
      id="webhooks"
      options={
        <Row>
          {canCreateWebhooks(workspaces).length > 0 && (
            <Button onClick={WebhookCreateModal.open}>New Webhook</Button>
          )}
        </Row>
      }
      title="Webhooks">
      {webhooks.length === 0 && !isLoading ? (
        <Message
          description={`Call external services when ${f_flat_runs ? 'searches' : 'experiments'} complete or throw errors.`}
          icon="webhooks"
          title="No Webhooks Registered"
        />
      ) : settings ? (
        <InteractiveTable<Webhook, Settings>
          columns={columns}
          containerRef={pageRef}
          dataSource={filteredWebhooks}
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
          settings={settings}
          showSorterTooltip={false}
          size="small"
          updateSettings={updateSettings}
          onChange={handleTableChange}
        />
      ) : (
        <SkeletonTable columns={columns.length} />
      )}
      <WebhookCreateModal.Component onSuccess={() => fetchWebhooks()} />
      <WebhookDeleteModal.Component webhook={selectedWebhook} onSuccess={() => fetchWebhooks()} />
      <WebhookEditModal.Component webhook={selectedWebhook} onSuccess={() => fetchWebhooks()} />
    </Page>
  );
};

export default WebhooksView;
