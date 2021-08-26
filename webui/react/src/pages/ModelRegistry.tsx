import { ColumnsType } from 'antd/lib/table';
import React, { useCallback, useMemo, useState } from 'react';

import Page from 'components/Page';
import ResponsiveTable from 'components/ResponsiveTable';
import Section from 'components/Section';
import { modelNameRenderer, relativeTimeRenderer } from 'components/Table';
import handleError, { ErrorType } from 'ErrorHandler';
import usePolling from 'hooks/usePolling';
import { getModels } from 'services/api';
import { V1GetModelsRequestSortBy } from 'services/api-ts-sdk';
import { ModelItem } from 'types';
import { isEqual } from 'utils/data';

const ModelRegistry: React.FC = () => {
  const [ models, setModels ] = useState<ModelItem[]>([]);

  const fetchModels = useCallback(async () => {
    try {
      const response = await getModels({});
      setModels(prev => {
        if (isEqual(prev, response.models)) return prev;
        return response.models;
      });
    } catch(e) {
      handleError({ message: 'Unable to fetch models.', silent: true, type: ErrorType.Api });
    }
  }, []);

  usePolling(fetchModels);

  const columns = useMemo(() => {
    const tableColumns: ColumnsType<ModelItem> = [
      { dataIndex: 'id', sorter: true, title: 'ID' },
      {
        dataIndex: 'name',
        key: V1GetModelsRequestSortBy.NAME,
        render: modelNameRenderer,
        sorter: true,
        title: 'Model name',
      },
      {
        dataIndex: 'description',
        key: V1GetModelsRequestSortBy.DESCRIPTION,
        sorter: true,
        title: 'Description',
      },
      { dataIndex: 'versions', title: 'Versions' },
      {
        dataIndex: 'lastUpdatedTime',
        key: V1GetModelsRequestSortBy.LASTUPDATEDTIME,
        render: relativeTimeRenderer,
        sorter: true,
        title: 'Last updated',
      },
      { dataIndex: 'labels', title: 'Labels' },
    ];

    return tableColumns;
  }, []);

  return (
    <Page docTitle="Model Registry" id="models">
      <Section title="Model Registry">
        <ResponsiveTable
          columns={columns}
          dataSource={models}
          showSorterTooltip={false} />
      </Section>
    </Page>
  );
};

export default ModelRegistry;
