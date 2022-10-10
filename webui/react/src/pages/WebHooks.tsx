// import { Button, Space } from 'antd';
import { SorterResult } from 'antd/lib/table/interface';
import React, { useCallback, useEffect, useMemo, useRef, useState } from 'react';

// import FilterCounter from 'components/FilterCounter';
// import InlineEditor from 'components/InlineEditor';
import InteractiveTable, { ColumnDef, InteractiveTableSettings } from 'components/InteractiveTable';
// import Link from 'components/Link';
import Page from 'components/Page';
import { defaultRowClassName } from 'components/Table';
// import TableFilterDropdown from 'components/TableFilterDropdown';
// import TableFilterSearch from 'components/TableFilterSearch';
import useSettings, { UpdateSettings } from 'hooks/useSettings';
// import { paths } from 'routes/utils';
import { getWebhooks } from 'services/api';
import Icon from 'shared/components/Icon/Icon';
import usePolling from 'shared/hooks/usePolling';
import { isEqual } from 'shared/utils/data';
import { ErrorType } from 'shared/utils/error';
// import { alphaNumericSorter } from 'shared/utils/sort';
import { Webhook } from 'types';
import handleError from 'utils/error';

import css from './Webhooks.module.scss';
import settingsConfig, { DEFAULT_COLUMN_WIDTHS, Settings } from './Webhooks.settings';

const WebhooksView: React.FC = () => {
  const [webhooks, setWebhooks] = useState<Webhook[]>([]);
  const [isLoading, setIsLoading] = useState(true);
  const [canceler] = useState(new AbortController());
  const pageRef = useRef<HTMLElement>(null);

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

  const columns = useMemo(() => {
    return [
      {
        dataIndex: 'id',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['id'],
        key: 'id',
        sorter: true,
        title: 'ID',
      },
      {
        dataIndex: 'url',
        defaultWidth: DEFAULT_COLUMN_WIDTHS['url'],
        key: 'url',
        sorter: true,
        title: 'URL',
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
        // tableLimit: tablePagination.pageSize,
        // tableOffset: (tablePagination.current - 1) * tablePagination.pageSize,
      };
      // const shouldPush = settings.tableOffset !== newSettings.tableOffset;
      updateSettings(newSettings, true);
    },
    [columns, settings.tableOffset, updateSettings],
  );

  useEffect(() => {
    return () => canceler.abort();
  }, [canceler]);

  return (
    <Page containerRef={pageRef} id="webhooks" title="Webhooks">
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
          rowClassName={defaultRowClassName({ clickable: false })}
          rowKey="name"
          settings={settings as InteractiveTableSettings}
          showSorterTooltip={false}
          size="small"
          updateSettings={updateSettings as UpdateSettings<InteractiveTableSettings>}
          onChange={handleTableChange}
        />
      )}
    </Page>
  );
};

export default WebhooksView;
