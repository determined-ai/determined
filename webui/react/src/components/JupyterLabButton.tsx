import Button from 'hew/Button';
import { shortcutToString } from 'hew/InputShortcut';
import { useModal } from 'hew/Modal';
import Tooltip from 'hew/Tooltip';
import React from 'react';

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
  const JupyterLabModal = useModal(JupyterLabModalComponent);
  const {
    settings: { jupyterLab: jupyterLabShortcut },
  } = useSettings<ShortcutSettings>(shortCutSettingsConfig);

  return (
    <div data-testid="jupyter-lab-button">
      {enabled ? (
        <>
          <Tooltip content={shortcutToString(jupyterLabShortcut)}>
            <Button onClick={JupyterLabModal.open}>Launch JupyterLab</Button>
          </Tooltip>
          <JupyterLabModal.Component workspace={workspace} />
        </>
      ) : (
        <Tooltip content="You do not have permission to launch JupyterLab" placement="leftBottom">
          <div>
            <Button disabled>Launch JupyterLab</Button>
          </div>
        </Tooltip>
      )}
    </div>
  );
};

export default JupyterLabButton;
