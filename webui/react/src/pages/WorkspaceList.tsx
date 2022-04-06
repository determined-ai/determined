import { Button } from 'antd';
import React, { useCallback } from 'react';

import Page from 'components/Page';
import useModalWorkspaceCreate from 'hooks/useModal/useModalWorkspaceCreate';

const WorkspaceList: React.FC = () => {
  const { modalOpen } = useModalWorkspaceCreate({});

  const handleWorkspaceCreateClick = useCallback(() => {
    modalOpen();
  }, [ modalOpen ]);
  return (
    <Page
      id="workspaces"
      options={<Button onClick={handleWorkspaceCreateClick}>New Workspace</Button>}
      title="Workspaces"
    />
  );
};

export default WorkspaceList;
