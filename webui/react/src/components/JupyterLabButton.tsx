import React from 'react';

import Button from 'components/kit/Button';
import Tooltip from 'components/kit/Tooltip';
import { Workspace } from 'types';

import JupyterLabModalComponent from './JupyterLabModal';
import { useModal } from './kit/Modal';

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
