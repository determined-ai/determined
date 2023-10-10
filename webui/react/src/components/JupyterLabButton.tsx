import Button from 'determined-ui/kit/Button';
import { shortcutToString } from 'determined-ui/kit/InputShortcut';
import { useModal } from 'determined-ui/kit/Modal';
import Tooltip from 'determined-ui/kit/Tooltip';
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
    <>
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
    </>
  );
};

export default JupyterLabButton;
