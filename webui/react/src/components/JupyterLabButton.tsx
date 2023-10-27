import Button from 'determined-ui/Button';
import { shortcutToString } from 'determined-ui/InputShortcut';
import { useModal } from 'determined-ui/Modal';
import Tooltip from 'determined-ui/Tooltip';
import React, { useRef } from 'react';

import JupyterLabModalComponent from 'components/JupyterLabModal';
import shortCutSettingsConfig, {
  Settings as ShortcutSettings,
} from 'components/UserSettings.settings';
import { useSettings } from 'hooks/useSettings';
import { Workspace } from 'types';
interface Props {
  enabled?: boolean;
  workspace?: Workspace;
}

const JupyterLabButton: React.FC<Props> = ({ enabled, workspace }: Props) => {
  const containerRef = useRef(null);
  const JupyterLabModal = useModal(JupyterLabModalComponent);
  const {
    settings: { jupyterLab: jupyterLabShortcut },
  } = useSettings<ShortcutSettings>(shortCutSettingsConfig, containerRef);

  return (
    <div ref={containerRef}>
      {enabled ? (
        <Tooltip content={shortcutToString(jupyterLabShortcut)}>
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
    </div>
  );
};

export default JupyterLabButton;
