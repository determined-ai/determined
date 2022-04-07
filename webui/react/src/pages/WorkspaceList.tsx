import { Button } from 'antd';
import React, { useCallback, useState } from 'react';

import InteractiveTable from 'components/InteractiveTable';
import Page from 'components/Page';
import { getFullPaginationConfig } from 'components/Table';
import useModalWorkspaceCreate from 'hooks/useModal/Workspace/useModalWorkspaceCreate';
import usePolling from 'hooks/usePolling';
import { getWorkspaces } from 'services/api';
import { Workspace } from 'types';
import handleError from 'utils/error';

const WorkspaceList: React.FC = () => {
  const { modalOpen } = useModalWorkspaceCreate({});
  const [ workspaces, setWorkspaces ] = useState<Workspace[]>([]);
  const [ isLoading, setIsLoading ] = useState(true);

  const handleWorkspaceCreateClick = useCallback(() => {
    modalOpen();
  }, [ modalOpen ]);

  const fetchWorkspaces = useCallback(async () => {
    try {
      const response = await getWorkspaces({});
      setWorkspaces(response.workspaces);
    } catch (e) {
      handleError(e, { publicSubject: 'Unable to fetch workspaces.' });
    } finally {
      setIsLoading(false);
    }
  }, []);

  usePolling(fetchWorkspaces);

  return (
    <Page
      id="workspaces"
      options={<Button onClick={handleWorkspaceCreateClick}>New Workspace</Button>}
      title="Workspaces">
      {/* <InteractiveTable
        areRowsSelected={!!settings.row}
        columns={columns}
        containerRef={pageRef}
        ContextMenu={ExperimentActionDropdown}
        dataSource={experiments}
        loading={isLoading}
        pagination={getFullPaginationConfig({
          limit: settings.tableLimit,
          offset: settings.tableOffset,
        }, total)}
        rowClassName={defaultRowClassName({ clickable: false })}
        rowKey="id"
        rowSelection={{
          onChange: handleTableRowSelect,
          preserveSelectedRowKeys: true,
          selectedRowKeys: settings.row ?? [],
        }}
        settings={settings as InteractiveTableSettings}
        showSorterTooltip={false}
        size="small"
        updateSettings={updateSettings as UpdateSettings<InteractiveTableSettings>}
      /> */}
    </Page>
  );
};

export default WorkspaceList;
