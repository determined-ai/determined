import React from 'react';

import JupyterLabModalComponent from 'components/JupyterLabModal';
import Button from 'components/kit/Button';
import { useModal } from 'components/kit/Modal';
import Tooltip from 'components/kit/Tooltip';
import { Workspace } from 'types';

interface Props {
  enabled?: boolean;
  workspace?: Workspace;
}

const JupyterLabButton: React.FC<Props> = ({ enabled, workspace }: Props) => {
  const JupyterLabModal = useModal(JupyterLabModalComponent);

  return (
    <>
      {enabled ? (
        <Button onClick={JupyterLabModal.open}>Launch JupyterLab</Button>
      ) : (
        <Tooltip placement="leftBottom" title="You do not have permission to launch JupyterLab">
          <div>
            <Button disabled>Launch JupyterLab</Button>
          </div>
        </Tooltip>
      )}
      <JupyterLabModal.Component workspace={workspace} />
    </>
  );
};

export default JupyterLabButton;
