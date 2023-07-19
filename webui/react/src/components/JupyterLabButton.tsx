import React from 'react';

import JupyterLabModalComponent from 'components/JupyterLabModal';
import Button from 'components/kit/Button';
import { useModal } from 'components/kit/Modal';
import Tooltip from 'components/kit/Tooltip';
import { useSettings } from 'hooks/useSettings';
import shortCutSettingsConfig, {
  Settings as ShortcutSettings,
} from 'pages/Admin/UserSettings.settings';
import { Workspace } from 'types';
import { shortcutToString } from 'utils/shortcut';
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
