import { Button, Dropdown, Menu, Modal } from 'antd';
import { ColumnsType } from 'antd/lib/table';
import React, { useCallback, useMemo, useState } from 'react';

import Icon from 'components/Icon';
import Page from 'components/Page';
import ResponsiveTable from 'components/ResponsiveTable';
import Section from 'components/Section';
import { modelNameRenderer, relativeTimeRenderer } from 'components/Table';
import { useStore } from 'contexts/Store';
import handleError, { ErrorType } from 'ErrorHandler';
import usePolling from 'hooks/usePolling';
import { getModels } from 'services/api';
import { V1GetModelsRequestSortBy } from 'services/api-ts-sdk';
import { ModelItem } from 'types';
import { isEqual } from 'utils/data';

import css from './ModelRegistry.module.scss';

const ModelRegistry: React.FC = () => {
  const { auth: { user } } = useStore();
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

  const deleteModel = useCallback((model: ModelItem) => {
    //send delete api request
  }, []);

  const switchArchived = useCallback((model: ModelItem) => {
    //check current archive status, switch it
  }, []);

  const showConfirmDelete = useCallback((model: ModelItem) => {
    Modal.confirm({
      closable: true,
      content: `Are you sure you want to delete this model "${model.name}" and all 
      of its versions from the model registry?`,
      icon: null,
      maskClosable: true,
      okText: 'Delete Model',
      okType: 'danger',
      onOk: () => deleteModel(model),
      title: 'Confirm Delete',
    });
  }, [ deleteModel ]);

  const columns = useMemo(() => {
    const overflowRenderer = (_:string, record: ModelItem) => {
      const isDeletable = user?.isAdmin;
      return (
        <Dropdown
          overlay={(
            <Menu>
              <Menu.Item
                onClick={() => switchArchived(record)}>
                  Archive
              </Menu.Item>
              <Menu.Item
                danger
                disabled={!isDeletable}
                onClick={() => showConfirmDelete(record)}>
                  Delete Model
              </Menu.Item>
            </Menu>
          )}>
          <Button className={css.overflow} type="text">
            <Icon name="overflow-vertical" size="tiny" />
          </Button>
        </Dropdown>
      );
    };

    const tableColumns: ColumnsType<ModelItem> = [
      { dataIndex: 'id', sorter: true, title: 'ID', width: 1 },
      {
        dataIndex: 'name',
        key: V1GetModelsRequestSortBy.NAME,
        render: modelNameRenderer,
        sorter: true,
        title: 'Model name',
        width: 250,
      },
      {
        dataIndex: 'description',
        key: V1GetModelsRequestSortBy.DESCRIPTION,
        sorter: true,
        title: 'Description',
      },
      { dataIndex: 'versions', title: 'Versions', width: 1 },
      {
        dataIndex: 'lastUpdatedTime',
        key: V1GetModelsRequestSortBy.LASTUPDATEDTIME,
        render: relativeTimeRenderer,
        sorter: true,
        title: 'Last updated',
        width: 1,
      },
      { dataIndex: 'tags', title: 'Tags' },
      { dataIndex: 'username', title: 'User', width: 1 },
      { render: overflowRenderer, title: '', width: 1 },
    ];

    return tableColumns;
  }, [ showConfirmDelete, switchArchived, user ]);

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
