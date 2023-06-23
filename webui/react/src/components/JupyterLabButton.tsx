import React from 'react';

import JupyterLabSettings from 'components/JupyterLab.settings';
import JupyterLabModalComponent from 'components/JupyterLabModal';
import Button from 'components/kit/Button';
import { useModal } from 'components/kit/Modal';
import Tooltip from 'components/kit/Tooltip';
import { useSettings } from 'hooks/useSettings';
import { Workspace } from 'types';
import { JupyterLabOptions } from 'utils/jupyter';
import { shortcutToString } from 'utils/shortcut';

interface Props {
  enabled?: boolean;
  workspace?: Workspace;
}

const JupyterLabButton: React.FC<Props> = ({ enabled, workspace }: Props) => {
  const JupyterLabModal = useModal(JupyterLabModalComponent);
  const { settings } = useSettings<JupyterLabOptions>(JupyterLabSettings);
  const shortcut = settings.shortcut;

  return (
    <>
      {enabled ? (
        <Tooltip content={shortcut && shortcutToString(shortcut)}>
          <Button onClick={JupyterLabModal.open}>Launch JupyterLab</Button>
        </Tooltip>
      ) : (
        <Tooltip content="You do not have permission to launch JupyterLab" placement="leftBottom">
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
