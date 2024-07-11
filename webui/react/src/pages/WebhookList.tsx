import { FilterValue, SorterResult, TablePaginationConfig } from 'antd/lib/table/interface';
import Button from 'hew/Button';
import Dropdown from 'hew/Dropdown';
import Icon from 'hew/Icon';
import Message from 'hew/Message';
import { useModal } from 'hew/Modal';
import Row from 'hew/Row';
import _ from 'lodash';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';

import Badge, { BadgeType } from 'components/Badge';
import Page from 'components/Page';
import InteractiveTable, { ColumnDef } from 'components/Table/InteractiveTable';
import SkeletonTable from 'components/Table/SkeletonTable';
import { defaultRowClassName, getFullPaginationConfig } from 'components/Table/Table';
import WebhookCreateModalComponent from 'components/WebhookCreateModal';
import WebhookDeleteModalComponent from 'components/WebhookDeleteModal';
import useFeature from 'hooks/useFeature';
import usePermissions from 'hooks/usePermissions';
import usePolling from 'hooks/usePolling';
import { useSettings } from 'hooks/useSettings';
import { paths } from 'routes/utils';
import { getWebhooks, testWebhook } from 'services/api';
import { V1Trigger, V1TriggerType } from 'services/api-ts-sdk/api';
import { Webhook } from 'types';
import handleError, { ErrorType } from 'utils/error';
import { alphaNumericSorter } from 'utils/sort';

import css from './WebhookList.module.scss';
import settingsConfig, { DEFAULT_COLUMN_WIDTHS, Settings } from './WebhookList.settings';

const MenuKey = {
  DeleteWebhook: 'delete-webhook',
  TestWebhook: 'test-webhook',
} as const;

const DROPDOWN_MENU = [
  { key: MenuKey.TestWebhook, label: 'Test Webhook' },
  { danger: true, key: MenuKey.DeleteWebhook, label: 'Delete Webhook' },
];

const WebhooksView: React.FC = () => {
  const [webhooks, setWebhooks] = useState<Webhook[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [canceler] = useState(new AbortController());
  const [selectedWebhook, setSelectedWebhook] = useState<Webhook>();
  const f_flat_runs = useFeature().isOn('flat_runs');

  const pageRef = useRef<HTMLElement>(null);

  const { canEditWebhooks } = usePermissions();

  const WebhookCreateModal = useModal(WebhookCreateModalComponent);
  const WebhookDeleteModal = useModal(WebhookDeleteModalComponent);

  const { settings, updateSettings } = useSettings<Settings>(settingsConfig);

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

  const handleDropdown = useCallback(
    async (key: string, record: Webhook) => {
      switch (key) {
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
      }
    },
    [WebhookDeleteModal],
  );

  const columns = useMemo(() => {
    const actionRenderer = (_: string, record: Webhook) => (
      <Dropdown menu={DROPDOWN_MENU} onClick={(key) => handleDropdown(key, record)}>
        <Button icon={<Icon name="overflow-vertical" title="Action menu" />} type="text" />
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
        if (t.triggerType === V1TriggerType.TASKLOG) {
          return (
            <li className={css.listBadge} key={t.id}>
              <Badge>TASKLOG</Badge>
            </li>
          );
        }
        return <></>;
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
  }, [f_flat_runs, handleDropdown]);

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
          {canEditWebhooks && <Button onClick={WebhookCreateModal.open}>New Webhook</Button>}
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
    </Page>
  );
};

export default WebhooksView;
