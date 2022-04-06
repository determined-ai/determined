import { Button } from 'antd';
import React, { useCallback } from 'react';
import { useParams } from 'react-router';

import Page from 'components/Page';
import useModalProjectCreate from 'hooks/useModal/useModalProjectCreate';

interface Params {
  workspaceId: string;
}

const WorkspaceDetails: React.FC = () => {
  const { workspaceId } = useParams<Params>();
  const { modalOpen } = useModalProjectCreate({ workspaceId: parseInt(workspaceId) });

  const handleProjectCreateClick = useCallback(() => {
    modalOpen();
  }, [ modalOpen ]);

  return (
    <Page
      id="workspaceDetails"
      options={<Button onClick={handleProjectCreateClick}>New Project</Button>}
      title="Workspace Details"
    />
  );
};

export default WorkspaceDetails;
